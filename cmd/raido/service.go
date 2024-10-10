package main

import (
	"context"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/fr13n8/raido/app"
	"github.com/fr13n8/raido/config"
	"github.com/kardianos/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
)

var (
	serviceName = "raido"
	serviceCmd  = &cobra.Command{
		Use:   "service",
		Short: "Service commands",
	}
)

func init() {
	serviceCmd.AddCommand(
		serviceStartCmd,
		serviceInstallCmd,
		serviceUninstallCmd,
		serviceRunCmd,
		serviceStopCmd,
		serviceRestartCmd,
		serviceStatusCmd,
	)
}

func newSVCConfig() *service.Config {
	return &service.Config{
		Name:        serviceName,
		DisplayName: "Raido",
		Description: "A “VPN-like” reverse proxy server with tunneling traffic through QUIC to access private network routes",
		Option:      make(service.KeyValue),
	}
}

func newSVC(prg *program, conf *service.Config) (service.Service, error) {
	s, err := service.New(prg, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create service: %w", err)
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
	log.Info().Msg("starting Raido service")

	split := strings.Split(serviceAddr, "://")
	// cleanup failed close
	stat, err := os.Stat(split[1])
	if err == nil && !stat.IsDir() {
		if err := os.Remove(split[1]); err != nil {
			log.Error().Msg("failed to remove existing socket file")
		}
	}

	listen, err := net.Listen(split[0], split[1])
	if err != nil {
		log.Error().Err(err).Msg("failed to listen service interface")
		return err
	}

	go func() {
		defer listen.Close()

		if err := os.Chmod(split[1], 0666); err != nil {
			log.Error().Msgf("failed setting service permissions: %v", split[1])
			return
		}

		s := app.NewServer(p.ctx, &config.ServiceServer{})

		log.Info().Msgf("starting service at %s ...", serviceAddr)
		if err := s.Run(listen); err != nil {
			log.Error().Err(err).Msg("failed to start service")
			return
		}

		log.Info().Msg("service stopped gracefully")
	}()

	return nil
}

func (p *program) Stop(srv service.Service) error {
	p.cancel()

	// Wait for the all services to stop
	time.Sleep(config.ShutdownTimeout)
	log.Info().Msg("service stopped")
	return nil
}
