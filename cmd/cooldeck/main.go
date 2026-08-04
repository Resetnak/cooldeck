package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/resetnak/cooldeck/internal/cli"
	"github.com/resetnak/cooldeck/internal/logging"
)

func main() {
	os.Exit(run())
}

func run() (exitCode int) {
	defer func() {
		if recovered := recover(); recovered != nil {
			_, _ = fmt.Fprintln(os.Stderr, "cooldeck crashed; run again with --debug for diagnostics")
			exitCode = 2
		}
	}()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := cli.Execute(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "error:", logging.Redact(err.Error()))
		return 1
	}
	return 0
}
