package main

import (
	"context"

	"github.com/fr13n8/raido/app"
	"github.com/fr13n8/raido/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	proxyCmd = &cobra.Command{
		Use:   "proxy",
		Short: "Proxy commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c := app.NewClient(cmd.Context(), &config.ServiceDialer{
				ServiceAddress: serviceAddr,
			})

			ctx := context.WithValue(cmd.Context(), app.ClientKey{}, c)
			cmd.SetContext(ctx)

			return nil
		},
	}

	proxyStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start proxy",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			certHash, err := c.ProxyStart(cmd.Context(), proxyAddr)
			if err != nil {
				log.Error().Err(err).Msg("failed to start proxy")
				return
			}

			log.Info().Msgf("proxy started with cert hash: %X", certHash)
		},
	}

	proxyStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop proxy",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			err := c.ProxyStop(cmd.Context())
			if err != nil {
				log.Error().Err(err).Msg("failed to stop proxy")
				return
			}

			log.Info().Msg("proxy stopped")
		},
	}
)

func init() {
	proxyStartCmd.Flags().StringVar(&proxyAddr, "proxy-addr", "0.0.0.0:8787", "Proxy listen address (e.g., :8787)")

	proxyCmd.AddCommand(proxyStartCmd)
	proxyCmd.AddCommand(proxyStopCmd)
}
