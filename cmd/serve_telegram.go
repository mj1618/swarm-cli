package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/mj1618/swarm-cli/internal/telegram"
	"github.com/spf13/cobra"
)

var (
	serveTelegramToken      string
	serveTelegramAllowUsers []string
	serveTelegramModel      string
	serveTelegramLabels     []string
	serveTelegramTimeout    string
	serveTelegramMaxConc    int
)

var serveTelegramCmd = &cobra.Command{
	Use:   "telegram",
	Short: "Serve as a Telegram bot that spawns agents for incoming messages",
	Long: `Start a Telegram bot that listens for incoming messages via long polling
and spawns agents to handle each message. The agent's output summary is sent
back to the chat.

The bot token can be provided via the --token flag or the TELEGRAM_BOT_TOKEN
environment variable.

At least one --allow-user must be specified. The bot will only respond to
messages from these Telegram usernames.`,
	Example: `  # Start bot with token from env var
  TELEGRAM_BOT_TOKEN=<token> swarm serve telegram --allow-user myusername

  # Start bot with explicit token and model
  swarm serve telegram --token <token> --allow-user alice --allow-user bob -m sonnet

  # With per-agent timeout and concurrency limit
  swarm serve telegram --allow-user myuser --timeout 5m --max-concurrent 3`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Resolve bot token
		token := serveTelegramToken
		if token == "" {
			token = os.Getenv("TELEGRAM_BOT_TOKEN")
		}
		if token == "" {
			return fmt.Errorf("bot token required: use --token flag or set TELEGRAM_BOT_TOKEN environment variable")
		}

		// Validate allow-user
		if len(serveTelegramAllowUsers) == 0 {
			return fmt.Errorf("at least one --allow-user is required")
		}

		// Strip leading @ from usernames
		for i, u := range serveTelegramAllowUsers {
			serveTelegramAllowUsers[i] = strings.TrimPrefix(u, "@")
		}

		// Resolve model
		model := serveTelegramModel
		if model == "" {
			model = appConfig.Model
		}

		// Parse timeout
		var agentTimeout time.Duration
		timeoutStr := serveTelegramTimeout
		if timeoutStr == "" {
			timeoutStr = "10m"
		}
		var err error
		agentTimeout, err = time.ParseDuration(timeoutStr)
		if err != nil {
			return fmt.Errorf("invalid timeout %q: %w", timeoutStr, err)
		}

		// Parse labels
		labels := make(map[string]string)
		for _, l := range serveTelegramLabels {
			parts := strings.SplitN(l, "=", 2)
			if len(parts) != 2 {
				return fmt.Errorf("invalid label format %q (expected key=value)", l)
			}
			labels[parts[0]] = parts[1]
		}

		// Get working directory
		workingDir, err := os.Getwd()
		if err != nil {
			return fmt.Errorf("failed to get working directory: %w", err)
		}

		// Create client and validate token
		client := telegram.NewClient(token)
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		bot, err := client.GetMe(ctx)
		if err != nil {
			return fmt.Errorf("failed to validate bot token: %w", err)
		}
		fmt.Fprintf(os.Stderr, "[telegram] Bot @%s started (model: %s, timeout: %s, max-concurrent: %d)\n",
			bot.Username, model, agentTimeout, serveTelegramMaxConc)
		fmt.Fprintf(os.Stderr, "[telegram] Allowed users: %s\n", strings.Join(serveTelegramAllowUsers, ", "))
		fmt.Fprintf(os.Stderr, "[telegram] Listening for messages...\n")

		// Set up signal handling
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

		// Create and start server
		server, err := telegram.NewServer(telegram.ServerConfig{
			Client:        client,
			AllowedUsers:  serveTelegramAllowUsers,
			Model:         model,
			Command:       appConfig.Command,
			AppConfig:     appConfig,
			WorkingDir:    workingDir,
			Scope:         GetScope(),
			Labels:        labels,
			AgentTimeout:  agentTimeout,
			MaxConcurrent: serveTelegramMaxConc,
		})
		if err != nil {
			return fmt.Errorf("failed to create server: %w", err)
		}

		// Run server in goroutine
		errChan := make(chan error, 1)
		go func() {
			errChan <- server.Run(ctx)
		}()

		// Wait for signal or error
		select {
		case sig := <-sigChan:
			fmt.Fprintf(os.Stderr, "\n[telegram] Received %v, shutting down...\n", sig)
			cancel()
			server.Shutdown(30 * time.Second)
		case err := <-errChan:
			if err != nil {
				return err
			}
		}

		return nil
	},
}

func init() {
	serveCmd.AddCommand(serveTelegramCmd)

	serveTelegramCmd.Flags().StringVar(&serveTelegramToken, "token", "", "Telegram bot token (or set TELEGRAM_BOT_TOKEN env var)")
	serveTelegramCmd.Flags().StringArrayVar(&serveTelegramAllowUsers, "allow-user", nil, "Allowed Telegram usernames (required, repeatable)")
	serveTelegramCmd.Flags().StringVarP(&serveTelegramModel, "model", "m", "", "Model override for agents")
	serveTelegramCmd.Flags().StringArrayVarP(&serveTelegramLabels, "label", "l", nil, "Labels for spawned agents (key=value)")
	serveTelegramCmd.Flags().StringVar(&serveTelegramTimeout, "timeout", "10m", "Per-agent timeout")
	serveTelegramCmd.Flags().IntVar(&serveTelegramMaxConc, "max-concurrent", 5, "Maximum concurrent agents")
}
