package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/fr13n8/raido/app"
	"github.com/fr13n8/raido/config"
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
	agentCmd = &cobra.Command{
		Use:   "agent",
		Short: "Agent commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c := app.NewClient(context.TODO(), &config.ServiceDialer{
				ServiceAddress: serviceAddr,
			})

			ctx := context.WithValue(cmd.Context(), app.ClientKey{}, c)
			cmd.SetContext(ctx)

			return nil
		},
	}

	agentListCmd = &cobra.Command{
		Use:   "list",
		Short: "List agents and their status",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			agents, err := c.GetAgents(cmd.Context())
			if err != nil {
				log.Error().Err(err).Msg("failed to get agents")
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

			for id, a := range agents {
				status := "Active"
				if !a.Status {
					status = "Inactive"
				}
				t.Row(id, a.Name, strings.Join(a.Routes, "\n"), status)
			}

			fmt.Println(t)
		},
	}

	agentStartTunnelCmd = &cobra.Command{
		Use:   "start-tunnel",
		Short: "Start tunnel to agent",
		Run: func(cmd *cobra.Command, args []string) {
			if agentId == "" {
				log.Error().Msg("agent ID is required")
				return
			}

			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("start tunnel...")
			if err := c.AgentTunnelStart(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("failed to start tunnel")
				return
			}

			log.Info().Msg("tunnel started")
		},
	}

	agentStopTunnelCmd = &cobra.Command{
		Use:   "stop-tunnel",
		Short: "Stop tunnel to agent",
		Run: func(cmd *cobra.Command, args []string) {
			if agentId == "" {
				log.Error().Msg("agent ID is required")
				return
			}

			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("stop tunnel...")
			if err := c.AgentTunnelStop(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("Failed to stop tunnel")
				return
			}

			log.Info().Msg("tunnel stopped")
		},
	}
)

func init() {
	// agentCmd.PersistentFlags().StringVar(&serviceAddr, "service-addr", "unix:///var/run/raido.sock", "Service listen address")

	agentStartTunnelCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID")
	agentStopTunnelCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID")

	agentCmd.AddCommand(agentListCmd, agentStartTunnelCmd, agentStopTunnelCmd)
}
