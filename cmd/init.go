package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new swarm project",
	Long:  `Initialize a new swarm project. Visit the starter repo for templates and setup instructions.`,
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	cyan := color.New(color.FgCyan, color.Bold)
	white := color.New(color.FgWhite, color.Bold)
	faint := color.New(color.Faint)

	cyan.Println("  ╭──────────────────────────────────────╮")
	cyan.Print("  │  ")
	white.Print("swarm init")
	faint.Print("  — project setup wizard")
	cyan.Println("  │")
	cyan.Println("  ╰──────────────────────────────────────╯")
	fmt.Println()
	fmt.Println("  To get started, clone the starter repo:")
	fmt.Println()
	white.Println("    https://github.com/mj1618/swarm-starter")
	fmt.Println()
	faint.Println("  It includes templates, PLAN.md, and swarm.yaml to get you up and running.")
	fmt.Println()

	return nil
}
