package main

import (
	"context"

	"github.com/kardianos/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	serviceStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start service",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), newSVCConfig())
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			if err := s.Start(); err != nil {
				log.Error().Err(err).Msg("failed to start service")
				return
			}

			log.Info().Msg("service started")
		},
	}

	serviceRunCmd = &cobra.Command{
		Use:   "run",
		Short: "Run service in foreground mode",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), newSVCConfig())
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			if err := s.Run(); err != nil {
				log.Error().Err(err).Msg("failed to run service")
				return
			}
		},
	}

	serviceStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop service",
		Run: func(cmd *cobra.Command, args []string) {
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

			log.Info().Msg("service stopped")
		},
	}

	serviceRestartCmd = &cobra.Command{
		Use:   "restart",
		Short: "Restart service",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), newSVCConfig())
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			if err := s.Restart(); err != nil {
				log.Error().Err(err).Msg("failed to restart service")
				return
			}

			log.Info().Msg("service restarted")
		},
	}

	serviceStatusCmd = &cobra.Command{
		Use:   "status",
		Short: "Service status",
		Run: func(cmd *cobra.Command, args []string) {
			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), newSVCConfig())
			if err != nil {
				log.Error().Err(err).Msg("failed to create service service")
				return
			}

			status, err := s.Status()
			if err != nil {
				log.Error().Err(err).Msg("failed to get service status")
				return
			}

			if status == service.StatusRunning {
				log.Info().Msg("service is running")
				return
			}
			if status == service.StatusStopped {
				log.Info().Msg("service is stopped")
				return
			}

			log.Error().Msg("service is in unknown state")
		},
	}
)
