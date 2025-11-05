package xml

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/bredtape/bulk_ops/archive"
	"github.com/pkg/errors"
)

// HandlePruneXPath which accepts a .zip archive with .xml files
// Each file have the matching XPaths pruned
// The returned body will contain a .zip archive with the pruned files
// A list of XPaths is provided by the (optionally repeated) query param 'xpath'.
func HandlePruneXPath() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		xpaths := r.URL.Query()["xpath"]
		if len(xpaths) == 0 {
			http.Error(w, "no 'xpath' query params specified", http.StatusBadRequest)
			return
		}

		if err := validateXPaths(xpaths); err != nil {
			http.Error(w, fmt.Sprintf("invalid XPaths specified: %v", err), http.StatusBadRequest)
			return
		}

		process := func(name string, w io.Writer, r io.Reader) error {
			return removeXPathNodes(xpaths, w, r)
		}

		defer r.Body.Close()
		err := archive.Process(w, r.Body, r.Header.Get("Content-Type"), r.ContentLength, process)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

func validateXPaths(xs []string) error {
	for _, x := range xs {
		if strings.TrimSpace(x) == "" {
			return errors.New("empty/blank")
		}
	}
	return nil
}

func removeXPathNodes(xpaths []string, w io.Writer, r io.Reader) error {
	doc, err := xmlquery.Parse(r)
	if err != nil {
		return errors.Wrap(err, "failed to parse XML")
	}

	// Remove nodes for each XPath expression
	for _, xpath := range xpaths {
		nodes, err := xmlquery.QueryAll(doc, xpath)
		if err != nil {
			return errors.Wrapf(err, "invalid XPath '%s'", xpath)
		}

		// Remove each matched node
		for _, node := range nodes {
			xmlquery.RemoveFromTree(node)
		}
	}

	// Convert back to XML string

	return doc.WriteWithOptions(w,
		xmlquery.WithOutputSelf(), // root node
		xmlquery.WithIndentation("  "),
		xmlquery.WithoutPreserveSpace(), // remove extra whitespace
	)
}
