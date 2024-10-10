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
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			log.Logger = log.Output(zerolog.ConsoleWriter{
				Out: os.Stderr,
				FormatTimestamp: func(i interface{}) string {
					return ""
				},
			})

			if verbose {
				zerolog.SetGlobalLevel(zerolog.TraceLevel)
			}

			return nil
		},
	}
)

func init() {
	rootCmd.PersistentFlags().BoolVar(&verbose, "v", false, "enable verbose mode")
	rootCmd.AddCommand(serviceCmd)
	rootCmd.AddCommand(agentCmd)
	rootCmd.AddCommand(proxyCmd)
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
		log.Error().Err(err).Msg("failed to execute command")
	}
}
