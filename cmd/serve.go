package cmd

import "github.com/spf13/cobra"

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run swarm as a long-running service with integrations",
	Long:  `Start swarm as a server that responds to messages from external services like Telegram.`,
}
