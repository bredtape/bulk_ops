package xml

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestHandlePruneXPath_withDescendants(t *testing.T) {
	runXMLSingleTest(t, "abc.xml", "abc_no_comment.xml", "//comment")
}

func TestHandlePruneXPath_withSpecificLevel(t *testing.T) {
	runXMLSingleTest(t, "blog.xml", "blog_no_content_comment.xml", "/blog/post/content/comment")
}

func runXMLSingleTest(t *testing.T, inputFile, expectedFile string, xpaths ...string) {
	t.Helper()

	_, archiveData, err := createArchiveFromTestdata(inputFile)
	assert.Nil(t, err)

	h := HandlePruneXPath()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "http://fake/it", bytes.NewReader(archiveData))
	req.Header.Set("Content-Type", "application/zip")
	query := req.URL.Query()
	for _, xpath := range xpaths {
		query.Add("xpath", xpath)
	}
	req.URL.RawQuery = query.Encode()
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	resp := rec.Result()
	defer resp.Body.Close()
	actual, err := io.ReadAll(resp.Body)
	assert.Nil(t, err)

	if !assert.Equal(t, 200, resp.StatusCode) {
		t.Logf("body: %s", string(actual))
		t.Errorf("should have status 200, but was %d", resp.StatusCode)
		return
	}

	t.Logf("response headers: %v", resp.Header)
	assert.Equal(t, resp.Header.Get("Content-Type"), "application/zip")

	actualFileData, err := unpackZipArchive(bytes.NewReader(actual))
	assert.Nil(t, err)

	expectedData, _, err := createArchiveFromTestdata(expectedFile)
	assert.Nil(t, err)

	expectedData[inputFile] = expectedData[expectedFile]
	delete(expectedData, expectedFile)

	compareFileData(t, expectedData, actualFileData)
}

func compareFileData(t *testing.T, expected, actual map[string][]byte) {
	t.Helper()

	seen := map[string]bool{}
	for k := range expected {
		seen[k] = true
	}
	for k := range actual {
		if !seen[k] {
			t.Errorf("actual does not have %s", k)
		}
	}
	if len(expected) != len(actual) {
		t.Errorf("expected count %d, actual %d", len(expected), len(actual))
	}

	for name := range expected {
		actualData, exists := actual[name]
		if !exists {
			t.Errorf("no actual data for file %s", name)
			continue
		}

		ed := string(expected[name])
		ad := string(actualData)

		if string(ad) != string(expected[name]) {
			t.Errorf("filename %s, expected:\n'%s'\nactual:\n'%s'\n", name, ed, ad)
		}
	}
}

func unpackZipArchive(r io.Reader) (map[string][]byte, error) {
	buf := bytes.Buffer{}
	_, err := io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}
	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		return nil, err
	}

	result := make(map[string][]byte, len(zr.File))
	for _, meta := range zr.File {
		rc, err := meta.Open()
		if err != nil {
			return nil, err
		}
		data, err := io.ReadAll(rc)
		if err != nil {
			return nil, err
		}
		result[meta.Name] = data
		err = rc.Close()
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

func createArchiveFromTestdata(files ...string) (fileData map[string][]byte, archiveData []byte, err error) {
	fileData = make(map[string][]byte, len(files))
	buf := bytes.Buffer{}
	zw := zip.NewWriter(&buf)

	for _, name := range files {
		fileWriter, err := zw.Create(name)
		if err != nil {
			return nil, nil, err
		}

		f, err := os.Open("testdata/" + name)
		if err != nil {
			return nil, nil, err
		}

		buf := bytes.Buffer{}
		_, err = io.Copy(&buf, f)
		if err != nil {
			return nil, nil, err
		}
		f.Close()

		data := buf.Bytes()
		fileData[name] = data

		n, err := io.Copy(fileWriter, bytes.NewReader(data))
		if err != nil {
			return nil, nil, err
		}
		if n == 0 {
			return nil, nil, fmt.Errorf("no data written for filename %s", name)
		}
	}

	err = zw.Close()
	if err != nil {
		return nil, nil, err
	}

	result := buf.Bytes()
	if len(result) == 0 {
		return nil, nil, errors.New("no resulting data")
	}

	return fileData, result, nil
}
