/*
Copyright © 2025 NAME HERE <EMAIL ADDRESS>
*/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/ravinald/wifimgr/cmd"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Second signal forces a hard exit. The first signal cancels ctx, which
	// unwinds long-running loops cleanly; the second is the escape hatch
	// for anything that doesn't honour cancellation.
	go func() {
		<-ctx.Done()
		hardCh := make(chan os.Signal, 1)
		signal.Notify(hardCh, os.Interrupt, syscall.SIGTERM)
		<-hardCh
		fmt.Fprintln(os.Stderr, "received second signal, exiting immediately")
		os.Exit(130)
	}()

	if err := cmd.Execute(ctx); err != nil {
		os.Exit(1)
	}
}
