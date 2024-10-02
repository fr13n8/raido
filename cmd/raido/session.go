package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/fr13n8/raido/config"
	"github.com/fr13n8/raido/service"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	re          = lipgloss.NewRenderer(os.Stdout)
	HeaderStyle = re.NewStyle().Bold(true).Align(lipgloss.Center)
	CellStyle   = re.NewStyle().Padding(0, 1)
	RowStyle    = CellStyle
	BorderStyle = lipgloss.NewStyle()
)

var (
	sessionCmd = &cobra.Command{
		Use:   "session",
		Short: "Session commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// send all this flags to the subcommands
			dAddr, err := cmd.Parent().Flags().GetString("service-addr")
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse service listen address")
				return err
			}
			certFile, err := cmd.Parent().Flags().GetString("cert-file")
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse cert file path")
				return err
			}
			keyFile, err := cmd.Parent().Flags().GetString("key-file")
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse key file path")
				return err
			}

			c, err := service.NewClient(context.TODO(), dAddr, &config.TLSConfig{
				CertFile:           certFile,
				KeyFile:            keyFile,
				ServerName:         "localhost",
				InsecureSkipVerify: true,
			})
			if err != nil {
				log.Error().Err(err).Msg("Failed to create client")
				return err
			}

			ctx := context.WithValue(cmd.Context(), service.ClientKey{}, c)
			cmd.SetContext(ctx)

			return nil
		},
	}

	sessionsListCmd = &cobra.Command{
		Use:   "list",
		Short: "List sessions",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(service.ClientKey{}).(*service.Client)

			sessions, err := c.GetSessions(cmd.Context())
			if err != nil {
				log.Error().Err(err).Msg("Failed to get sessions")
				return
			}

			t := table.New().
				Border(lipgloss.NormalBorder()).
				BorderStyle(BorderStyle).
				StyleFunc(func(row, col int) lipgloss.Style {
					if row == 0 {
						return HeaderStyle
					}

					return RowStyle
				}).
				Headers("ID", "Hostname", "Routes", "Status")

			for id, a := range sessions {
				status := "Active"
				if !a.Status {
					status = "Inactive"
				}
				t.Row(id, a.Name, strings.Join(a.Routes, "\n"), status)
			}

			fmt.Println(t)
		},
	}

	sessionStartTunnelCmd = &cobra.Command{
		Use:   "start-tunnel",
		Short: "Start tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			sessionId, err := cmd.Flags().GetString("session-id")
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse session ID")
				return
			}

			c := cmd.Context().Value(service.ClientKey{}).(*service.Client)

			log.Info().Msg("Start tunnel...")
			if err := c.SessionTunnelStart(cmd.Context(), sessionId); err != nil {
				log.Error().Err(err).Msg("Failed to start tunnel")
				return
			}

			log.Info().Msg("Tunnel started")
		},
	}

	sessionStopTunnelCmd = &cobra.Command{
		Use:   "stop-tunnel",
		Short: "Stop tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			sessionId, err := cmd.Flags().GetString("session-id")
			if err != nil {
				log.Error().Err(err).Msg("Failed to parse session ID")
				return
			}

			c := cmd.Context().Value(service.ClientKey{}).(*service.Client)

			log.Info().Msg("Stop tunnel...")
			if err := c.SessionTunnelStop(cmd.Context(), sessionId); err != nil {
				log.Error().Err(err).Msg("Failed to stop tunnel")
				return
			}

			log.Info().Msg("Tunnel stopped")
		},
	}
)

func init() {
	sessionCmd.Flags().StringP("service-addr", "d", "https://127.0.0.1:6660", "Service listen address (e.g., https://:6660)")
	sessionCmd.Flags().StringP("cert-file", "c", "cert.crt", "TLS certificate file path")
	sessionCmd.Flags().StringP("key-file", "k", "cert.key", "TLS private key file path")

	sessionStartTunnelCmd.Flags().StringP("session-id", "s", "", "Session ID")
	sessionStopTunnelCmd.Flags().StringP("session-id", "s", "", "Session ID")

	sessionCmd.AddCommand(sessionsListCmd, sessionStartTunnelCmd, sessionStopTunnelCmd)
}
