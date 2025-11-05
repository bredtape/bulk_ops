package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"

	bulkops "github.com/bredtape/bulk_ops"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()

	err := bulkops.Run(ctx, os.Args, os.Getenv, os.Stdin, os.Stdout, os.Stderr)
	if err != nil {
		slog.Error("run failed", "err", err)
		defer os.Exit(1) // to have context cancel() called before Exit
		return
	}
	slog.Info("exited with no error")
}
