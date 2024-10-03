package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/fr13n8/raido/proxy/transport/quic"

	"github.com/fr13n8/raido/config"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	flagSet := flag.NewFlagSet("agent", flag.ExitOnError)
	proxyAddress := flagSet.String("pa", "", "relay address to connect to (e.g., 192.168.100.7:3333)")
	verbose := flagSet.Bool("v", false, "enable verbose mode")
	insecureSkipVerify := flagSet.Bool("sv", true, "skip TLS certficate verification")
	// CACertificatePath := flagSet.String("cp", "", "path to TLS CA certificate PEM file")

	flagSet.Usage = func() {
		fmt.Fprintln(os.Stderr, `Start agent.

Usage:
agent [flags]

Flags:`)
		flagSet.PrintDefaults()
		fmt.Fprintln(os.Stderr)
	}

	if err := flagSet.Parse(os.Args[1:]); err != nil {
		return
	}

	log.Logger = log.Output(zerolog.ConsoleWriter{
		Out: os.Stderr,
		FormatTimestamp: func(i interface{}) string {
			return ""
		},
		// TimeFormat: time.RFC3339,
	})

	if *verbose {
		zerolog.SetGlobalLevel(zerolog.TraceLevel)
	}

	if *proxyAddress == "" {
		log.Fatal().Msg("please, specify the relay server listen address -pa host:port")
	}

	tlsConfig := &config.TLSConfig{
		// CAFile:             *CACertificatePath,
		ServerName:         "localhost",
		InsecureSkipVerify: *insecureSkipVerify,
	}

	dialerConf := &config.Dialer{
		ProxyAddress: *proxyAddress,
		TLSConfig:    tlsConfig,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	d, err := quic.NewDialer(ctx, dialerConf)
	if err != nil {
		log.Fatal().Err(err).Msg("create dialer error")
	}

	// go func() {
	// 	http.Handle("/prometheus", promhttp.Handler())
	// 	log.Fatal().Err(http.ListenAndServe("0.0.0.0:5001", nil)).Send()
	// }()

	if err := d.Run(ctx); err != nil {
		log.Error().Err(err).Send()
		return
	}

	log.Info().Msg("agent stopped")
}
