package main

import (
	"context"

	"github.com/kardianos/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	serviceInstallCmd = &cobra.Command{
		Use:   "install",
		Short: "Install service",
		Run: func(cmd *cobra.Command, args []string) {
			log.Info().Msg("Service installed")
			svcConfig := newSVCConfig()

			// svcConfig.Arguments = []string{
			// 	"service",
			// 	"run",
			// 	configPath,
			// 	"--log-level",
			// 	logLevel,
			// 	"--service-addr",
			// 	daemonAddr,
			// }

			ctx, cancel := context.WithCancel(cmd.Context())
			s, err := newSVC(newProgram(ctx, cancel), svcConfig)
			if err != nil {
				cmd.PrintErrln(err)
				return
			}

			err = s.Install()
			if err != nil {
				cmd.PrintErrln(err)
				return
			}

			log.Info().Msg("service successfully installed")
		},
	}
)

func newSVCConfig() *service.Config {
	return &service.Config{
		Name:        "raido",
		DisplayName: "Raido",
		Option:      make(service.KeyValue),
	}
}

func newSVC(prg *program, conf *service.Config) (service.Service, error) {
	s, err := service.New(prg, conf)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create service service")
		return nil, err
	}
	return s, nil
}

type program struct {
	ctx    context.Context
	cancel context.CancelFunc
	serv   *grpc.Server
}

func newProgram(ctx context.Context, cancel context.CancelFunc) *program {
	return &program{ctx: ctx, cancel: cancel}
}

func (p *program) Start(svc service.Service) error {
	return nil
}

func (p *program) Stop(srv service.Service) error {
	return nil
}
