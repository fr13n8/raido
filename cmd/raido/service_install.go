package main

import (
	"context"
	"runtime"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	serviceInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install service",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("installing service")
			svcConfig := newSVCConfig()

			svcConfig.Arguments = []string{
				"service",
				"run",
			}

			if runtime.GOOS == "linux" {
				// Respected only by systemd systems
				svcConfig.Dependencies = []string{"After=network.target syslog.target"}
			}
			if runtime.GOOS == "windows" {
				svcConfig.Option["OnFailure"] = "restart"
			}

			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), svcConfig)
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			if err := s.Install(); err != nil {
				log.Error().Err(err).Msg("failed to install service")
				return
			}

			log.Info().Msg("service successfully installed")
		},
	}
)
