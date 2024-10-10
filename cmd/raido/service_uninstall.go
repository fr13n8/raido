package main

import (
	"context"

	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	serviceUninstallCmd = &cobra.Command{
		Use:   "uninstall",
		Short: "Uninstall service",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("unistalling service")

			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), newSVCConfig())
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			if err := s.Stop(); err != nil {
				log.Error().Err(err).Msg("failed to stop service")
				return
			}

			if err := s.Uninstall(); err != nil {
				log.Error().Err(err).Msg("failed to uninstall service")
				return
			}

			log.Info().Msg("service successfully uninstalled")
		},
	}
)
