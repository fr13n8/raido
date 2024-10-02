package service

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"connectrpc.com/connect"
	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/service/proto"
	"github.com/fr13n8/raido/service/proto/servicev1connect"
	"github.com/quic-go/quic-go/http3"
	"google.golang.org/grpc/status"
)

type ClientKey struct{}

type Client struct {
	serviceClient servicev1connect.RaidoServiceClient
	qConn         *http3.RoundTripper
}

func NewClient(ctx context.Context, url string, cfg *config.TLSConfig) (*Client, error) {
	tlsConf := &tls.Config{
		MinVersion:         tls.VersionTLS13,
		NextProtos:         []string{"raido-service"},
		ServerName:         cfg.ServerName,
		InsecureSkipVerify: cfg.InsecureSkipVerify,
	}

	tlsConf.RootCAs, _ = x509.SystemCertPool()
	if tlsConf.RootCAs == nil {
		tlsConf.RootCAs = x509.NewCertPool()
	}

	if cfg.CertFile != "" {
		caCertRaw, err := os.ReadFile(cfg.CertFile)
		if err != nil {
			return nil, err
		}

		if !tlsConf.RootCAs.AppendCertsFromPEM(caCertRaw) {
			return nil, fmt.Errorf("failed to append cert at path: %q", cfg.CertFile)
		}
	}

	roundTripper := &http3.RoundTripper{
		TLSClientConfig: tlsConf,
	}

	client := &http.Client{
		Transport: roundTripper,
		// Timeout:   time.Second * 5,
	}

	dClient := servicev1connect.NewRaidoServiceClient(client, url, connect.WithGRPC())

	return &Client{
		serviceClient: dClient,
	}, nil
}

func (c *Client) GetSessions(ctx context.Context) (map[string]*servicev1.Session, error) {
	resp, err := c.serviceClient.GetSessions(ctx, &connect.Request[servicev1.GetSessionsRequest]{})
	if err != nil {
		return nil, fmt.Errorf("failed to list sessions: %v", status.Convert(err).Message())
	}

	return resp.Msg.GetSessions(), nil
}

func (c *Client) SessionTunnelStart(ctx context.Context, sessionId string) error {
	_, err := c.serviceClient.SessionTunnelStart(ctx, &connect.Request[servicev1.SessionTunnelStartRequest]{
		Msg: &servicev1.SessionTunnelStartRequest{
			SessionId: sessionId,
		},
	})
	if err != nil {
		return err
	}

	return nil
}

func (c *Client) SessionTunnelStop(ctx context.Context, sessionId string) error {
	_, err := c.serviceClient.SessionTunnelStop(ctx, &connect.Request[servicev1.SessionTunnelStopRequest]{
		Msg: &servicev1.SessionTunnelStopRequest{
			SessionId: sessionId,
		},
	})
	if err != nil {
		return err
	}

	return nil
}
