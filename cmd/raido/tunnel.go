package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/fr13n8/raido/app"
	"github.com/fr13n8/raido/config"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var (
	tunnelCmd = &cobra.Command{
		Use:   "tunnel",
		Short: "Tunnel commands",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			c := app.NewClient(context.TODO(), &config.ServiceDialer{
				ServiceAddress: serviceAddr,
			})

			ctx := context.WithValue(cmd.Context(), app.ClientKey{}, c)
			cmd.SetContext(ctx)

			return nil
		},
	}

	tunnelListCmd = &cobra.Command{
		Use:   "list",
		Short: "List tunnels and their status",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			tunnels, err := c.TunnelList(cmd.Context())
			if err != nil {
				log.Error().Err(err).Msg("failed to get tunnels")
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
				Headers("№", "Agent ID", "Interface", "Routes", "Status")

			for id, tunnel := range tunnels {
				t.Row(fmt.Sprintf("%d", id+1), tunnel.AgentId, tunnel.Interface, strings.Join(tunnel.Routes, "\n"), tunnel.Status)
			}

			fmt.Println(t)
		},
	}

	tunnelStartCmd = &cobra.Command{
		Use:   "start",
		Short: "Start tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("start tunnel...")
			if err := c.TunnelStart(cmd.Context(), agentId, routes); err != nil {
				log.Error().Err(err).Msg("failed to start tunnel")
				return
			}

			log.Info().Msg("tunnel started")
		},
	}

	tunnelStopCmd = &cobra.Command{
		Use:   "stop",
		Short: "Stop tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("stop tunnel...")
			if err := c.TunnelStop(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("Failed to stop tunnel")
				return
			}

			log.Info().Msg("tunnel stopped")
		},
	}

	tunnelAddRouteCmd = &cobra.Command{
		Use:   "add-route",
		Short: "Add route to tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("add route to tunnel...")
			if err := c.TunnelAddRoute(cmd.Context(), agentId, routes); err != nil {
				log.Error().Err(err).Msg("failed to add route to tunnel")
				return
			}

			log.Info().Msg("route added to tunnel")
		},
	}

	tunnelRemoveRouteCmd = &cobra.Command{
		Use:   "remove-route",
		Short: "Remove route from tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("remove route from tunnel...")
			if err := c.TunnelRemoveRoute(cmd.Context(), agentId, routes); err != nil {
				log.Error().Err(err).Msg("failed to remove route from tunnel")
				return
			}

			log.Info().Msg("route removed from tunnel")
		},
	}

	tunnelPauseCmd = &cobra.Command{
		Use:   "pause",
		Short: "Pause tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("pause tunnel...")
			if err := c.TunnelPause(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("failed to pause tunnel")
				return
			}

			log.Info().Msg("tunnel paused")
		},
	}

	tunnelResumeCmd = &cobra.Command{
		Use:   "resume",
		Short: "Resume tunnel",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			log.Info().Msg("resume tunnel...")
			if err := c.TunnelResume(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("failed to resume tunnel")
				return
			}

			log.Info().Msg("tunnel resumed")
		},
	}
)

func init() {
	tunnelStartCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID for starting tunnel")
	tunnelStartCmd.MarkFlagRequired("agent-id")
	tunnelStartCmd.Flags().StringArrayVar(&routes, "routes", nil, "Routes to tunnel (e.g., 10.1.0.2/16,10.2.0.2/32,10.3.0.2/24)\nIf not provided, all routes will be tunneled")

	tunnelStopCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID for stopping tunnel")
	tunnelStopCmd.MarkFlagRequired("agent-id")

	tunnelAddRouteCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID to add route to tunnel")
	tunnelAddRouteCmd.MarkFlagRequired("agent-id")
	tunnelAddRouteCmd.Flags().StringArrayVar(&routes, "routes", nil, "Routes to tunnel (e.g., 10.1.0.2/16,10.2.0.2/32,10.3.0.2/24)")
	tunnelAddRouteCmd.MarkFlagRequired("routes")

	tunnelRemoveRouteCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID to remove route from tunnel")
	tunnelRemoveRouteCmd.MarkFlagRequired("agent-id")
	tunnelRemoveRouteCmd.Flags().StringArrayVar(&routes, "routes", nil, "Routes to tunnel (e.g., 10.1.0.2/16,10.2.0.2/32,10.3.0.2/24)")
	tunnelRemoveRouteCmd.MarkFlagRequired("routes")

	tunnelPauseCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID to pause tunnel")
	tunnelPauseCmd.MarkFlagRequired("agent-id")

	tunnelResumeCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID to resume tunnel")
	tunnelResumeCmd.MarkFlagRequired("agent-id")

	tunnelCmd.AddCommand(
		tunnelStartCmd,
		tunnelStopCmd,
		tunnelAddRouteCmd,
		tunnelRemoveRouteCmd,
		tunnelListCmd,
		tunnelPauseCmd,
		tunnelResumeCmd,
	)
}
