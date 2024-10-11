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
		Short: "List agents",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			agents, err := c.AgentList(cmd.Context())
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
				Headers("№", "ID", "Hostname", "Routes")

			i := 1
			for id, a := range agents {
				t.Row(fmt.Sprintf("%d", i), id, a.Name, strings.Join(a.Routes, "\n"))
			}

			fmt.Println(t)
		},
	}

	agentRemoveCmd = &cobra.Command{
		Use:   "remove",
		Short: "Remove agent",
		Run: func(cmd *cobra.Command, args []string) {
			c := cmd.Context().Value(app.ClientKey{}).(*app.Client)

			if err := c.AgentRemove(cmd.Context(), agentId); err != nil {
				log.Error().Err(err).Msg("failed to remove agent")
				return
			}

			log.Info().Msg("agent successfully removed")
		},
	}
)

func init() {
	agentRemoveCmd.Flags().StringVar(&agentId, "agent-id", "", "Agent ID to remove")
	agentRemoveCmd.MarkFlagRequired("agent-id")

	agentCmd.AddCommand(
		agentListCmd,
		agentRemoveCmd,
	)
}
