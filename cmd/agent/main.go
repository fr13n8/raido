package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"crypto/x509"
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/fr13n8/raido/proxy/protocol"
	"github.com/fr13n8/raido/proxy/transport"
	"github.com/fr13n8/raido/proxy/transport/core"
	"github.com/fr13n8/raido/proxy/transport/quic"
	"github.com/fr13n8/raido/proxy/transport/tcp"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	flagSet := flag.NewFlagSet("agent", flag.ExitOnError)
	proxyAddress := flagSet.String("pa", "", "relay address to connect to (e.g., 192.168.100.7:3333)")
	insecureSkipVerify := flagSet.Bool("isk", false, "skip TLS certficate verification")
	certHash := flagSet.String("ch", "", "certificate hash for accepting self-signed certificates")
	transportProtocol := flagSet.String("tp", "quic", "transport protocol (quic, tcp)")

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

	if *proxyAddress == "" {
		log.Fatal().Msg("please, specify the proxy server listen address -pa host:port")
	}

	host, _, err := net.SplitHostPort(*proxyAddress)
	if err != nil {
		log.Fatal().Err(err).Msg("invalid proxy address, please use host:port")
	}
	tlsConfig := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{protocol.Name},
		ServerName:         host,
		InsecureSkipVerify: *insecureSkipVerify,
	}
	if *certHash != "" {
		tlsConfig.InsecureSkipVerify = true
		tlsConfig.VerifyPeerCertificate = func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
			crtFingerprint := sha256.Sum256(rawCerts[0])
			crtMatch, err := hex.DecodeString(*certHash)
			if err != nil {
				return fmt.Errorf("failed to decode certificate hash: %w", err)
			}
			if !bytes.Equal(crtMatch, crtFingerprint[:]) {
				return fmt.Errorf("certificate hash mismatch %x != %x", crtMatch, crtFingerprint[:])
			}
			return nil
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	go func() {
		<-ctx.Done()
		stop()
	}()

	var transportImpl transport.Transport
	switch *transportProtocol {
	case "quic":
		transportImpl = quic.NewQUICTransport(tlsConfig)
	case "tcp":
		transportImpl = tcp.NewTCPTransport(tlsConfig)
	default:
		log.Fatal().Msgf("unsupported transport protocol: %s", *transportProtocol)
	}

	d := core.NewDialer(ctx, transportImpl, *proxyAddress)

	// go func() {
	// 	http.Handle("/prometheus", promhttp.Handler())
	// 	log.Fatal().Err(http.ListenAndServe("0.0.0.0:5001", nil)).Send()
	// }()

	if err := d.Run(ctx); err != nil {
		log.Error().Err(err).Msgf("failed to run dialer")
		return
	}

	log.Info().Msg("agent stopped")
}
