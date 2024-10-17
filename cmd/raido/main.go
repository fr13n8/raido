package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	rootCmd = &cobra.Command{
		Use:          "raido",
		SilenceUsage: true,
	}
)

func init() {
	rootCmd.AddCommand(
		serviceCmd,
		agentCmd,
		tunnelCmd,
		proxyCmd,
	)
}

func main() {
	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stderr,
		FormatTimestamp: func(i interface{}) string {
			return ""
		},
	})

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	if err := rootCmd.ExecuteContext(ctx); err != nil {
		os.Exit(1)
	}
}
