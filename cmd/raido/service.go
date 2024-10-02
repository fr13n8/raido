package main

import (
	"github.com/fr13n8/raido/config"
	proxy "github.com/fr13n8/raido/proxy/transport/quic"
	"github.com/fr13n8/raido/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	serviceCmd = &cobra.Command{
		Use:   "service",
		Short: "Service commands",
	}

	serviceStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start service",
		Run: func(cmd *cobra.Command, args []string) {
			dAddr, err := cmd.Flags().GetString("service-addr")
			if err != nil {
				log.Error().Err(err).Msg("failed to parse service listen address")
				return
			}
			pAddr, err := cmd.Flags().GetString("proxy-addr")
			if err != nil {
				log.Error().Err(err).Msg("failed to parse proxy listen address")
				return
			}
			certFile, err := cmd.Flags().GetString("cert-file")
			if err != nil {
				log.Error().Err(err).Msg("failed to parse cert file path")
				return
			}
			keyFile, err := cmd.Flags().GetString("key-file")
			if err != nil {
				log.Error().Err(err).Msg("failed to parse key file path")
				return
			}

			sCfg := &config.ServiceServer{
				Address: dAddr,
				TLSConfig: &config.TLSConfig{
					CertFile:   certFile,
					KeyFile:    keyFile,
					ServerName: "localhost",
				},
			}

			pCfg := &config.ProxyServer{
				Address: pAddr,
				TLSConfig: &config.TLSConfig{
					CertFile:   certFile,
					KeyFile:    keyFile,
					ServerName: "localhost",
				},
			}

			s, err := service.NewServer(sCfg)
			if err != nil {
				log.Error().Err(err).Msg("failed to create service")
				return
			}

			ps, err := proxy.NewServer(pCfg)
			if err != nil {
				log.Error().Err(err).Msg("failed to create proxy server")
				return
			}

			go func() {
				log.Info().Msg("starting proxy...")
				if err := ps.Listen(cmd.Context()); err != nil {
					log.Fatal().Err(err).Msg("Failed to listen proxy")
					return
				}
				log.Info().Msg("proxy server stopped")
			}()

			log.Info().Msg("starting service...")
			if err := s.Run(cmd.Context()); err != nil {
				log.Error().Err(err).Msg("failed to start service")
				return
			}
		},
	}

	serviceInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install service",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Service installed")
		},
	}

	serviceHealthCheckCmd = &cobra.Command{
		Use:   "ping",
		Short: "Health check",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("service is healthy")
		},
	}
)

func init() {
	serviceStartCmd.Flags().StringP("service-addr", "d", "127.0.0.1:6660", "Service listen address (e.g., :6660)")
	serviceStartCmd.Flags().StringP("proxy-addr", "p", "0.0.0.0:8787", "Proxy listen address (e.g., :8787)")
	serviceStartCmd.Flags().StringP("cert-file", "c", "cert.crt", "TLS certificate file path")
	serviceStartCmd.Flags().StringP("key-file", "k", "cert.key", "TLS private key file path")

	serviceCmd.AddCommand(serviceStartCmd, serviceHealthCheckCmd)
}
