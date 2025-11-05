package bulk_ops

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/peterbourgon/ff/v3"
	"github.com/pkg/errors"
)

const app = "bulk_ops"

type Config struct {
	HTTPAddress string
}

func Run(
	ctx context.Context,
	args []string,
	getenv func(string) string,
	stdin io.Reader,
	stdout io.Writer,
	stderr io.Writer) error {

	config, err := readArgs(args, stderr)
	if err != nil {
		return err
	}

	log := slog.With("httpAddress", config.HTTPAddress)

	mux := http.NewServeMux()
	err = registerRoutes(mux)
	if err != nil {
		return errors.Wrap(err, "failed to register routes")
	}

	server := &http.Server{
		Addr:    config.HTTPAddress,
		Handler: mux}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		log.Debug("begin shutdown server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "err", err)
		}
	}()

	log.Info("starting http server")
	if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func readArgs(args []string, stderr io.Writer) (Config, error) {
	var c Config

	envPrefix := strings.ToUpper(app)
	fs := flag.NewFlagSet(args[0], flag.ContinueOnError)
	fs.Usage = func() {
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "Options may also be set from the environment. Prefix with %s_, use all caps and replace any - with _\n", envPrefix)
	}

	fs.StringVar(&c.HTTPAddress, "http-addr", ":8600", "Listen address for HTTP service")

	slogOpt := slog.HandlerOptions{}
	var logLevel slog.Level
	fs.TextVar(&logLevel, "log-level", slog.LevelInfo, "Log level {DEBUG, INFO, WARN, ERROR}. Log level may also be set to a relative level or an integer, e.g. 'DEBUG-3', '6'")
	var logJSON bool
	fs.BoolVar(&logJSON, "log-json", true, "Log output as JSON")
	fs.BoolVar(&slogOpt.AddSource, "log-source", false, "Log output with source code information")
	var help bool
	fs.BoolVar(&help, "help", false, "Command line help. If enabled, will print options and exit")

	err := ff.Parse(fs, args[1:], ff.WithEnvVarPrefix(envPrefix), ff.WithEnvVarSplit("\n"))
	if err != nil {
		return c, errors.Wrap(err, "failed to parse command line options")
	}

	if help {
		fs.Usage()
	}

	slogOpt.Level = logLevel
	if logJSON {
		slog.SetDefault(slog.New(slog.NewJSONHandler(stderr, &slogOpt)))
	} else {
		slog.SetDefault(slog.New(slog.NewTextHandler(stderr, &slogOpt)))
	}

	return c, nil
}
