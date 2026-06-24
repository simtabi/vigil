// Package cmd implements the mta command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagScope  string
	flagConfig string
	flagJSON   bool
)

var rootCmd = &cobra.Command{
	Use:   "mta",
	Short: "Keep Microsoft Teams active on a configurable schedule",
	Long: `mta (ms-teams-activity) keeps Microsoft Teams showing as Available on a
configurable work schedule (e.g. Mon-Fri 08:00-17:00) or at will, using either
synthetic input or the Microsoft Graph presence API.

Run without a subcommand on a terminal to open the interactive TUI.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, _ []string) error {
		if term.IsTerminal(int(os.Stdin.Fd())) {
			return runTUI()
		}
		return cmd.Help()
	},
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func init() {
	pf := rootCmd.PersistentFlags()
	pf.StringVar(&flagScope, "scope", "user", "config/runtime scope: user|system")
	pf.StringVar(&flagConfig, "config", "", "path to config.json (overrides scope default)")
	pf.BoolVar(&flagJSON, "json", false, "emit machine-readable JSON where supported")
}

// scope resolves the --scope flag into a config.Scope, defaulting to user.
func scope() config.Scope {
	s := config.Scope(flagScope)
	if !s.Valid() {
		return config.ScopeUser
	}
	return s
}

// configPath returns the effective config path for the current scope/flags.
func configPath() (string, error) {
	if flagConfig != "" {
		return flagConfig, nil
	}
	return config.ConfigPath(scope())
}

// runtimeDir returns the runtime directory for the current scope.
func runtimeDir() (string, error) {
	return config.RuntimeDir(scope())
}

// loadConfig loads and validates the effective config.
func loadConfig() (config.Config, error) {
	p, err := configPath()
	if err != nil {
		return config.Config{}, err
	}
	c, err := config.Load(p)
	if os.IsNotExist(err) {
		return config.Config{}, fmt.Errorf("no config at %s — run `mta config init` first", p)
	}
	return c, err
}
