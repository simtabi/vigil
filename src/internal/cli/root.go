// Package cmd implements the vigil command-line interface.
package cli

import (
	"fmt"
	"os"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

var (
	flagScope   string
	flagConfig  string
	flagJSON    bool
	flagVerbose bool
	flagYes     bool
	flagNoInput bool
	flagNoColor bool
)

var rootCmd = &cobra.Command{
	Use:   "vigil",
	Short: "Keep Microsoft Teams active on a configurable schedule",
	Long: `vigil keeps Microsoft Teams showing as Available on a
configurable work schedule (e.g. Mon-Fri 08:00-17:00) or at will, using either
synthetic input or the Microsoft Graph presence API.

Run without a subcommand on a terminal to open the interactive TUI.

Global flags: --yes/-y (assume yes), --no-input (never prompt; for scripts),
--no-color, --json, --scope user|system. Honored env vars: NO_COLOR, TERM,
EDITOR (config edit), XDG_CONFIG_HOME/XDG_STATE_HOME (file locations).`,
	Example: `  vigil                      # open the interactive TUI
  vigil install --init       # configure + install + start the daemon
  vigil on --for 2h          # stay Available for two hours
  vigil status --json        # machine-readable status
  vigil upgrade --check      # see if a newer release exists`,
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
	pf.BoolVar(&flagVerbose, "verbose", false, "verbose (debug) logging")
	pf.BoolVarP(&flagYes, "yes", "y", false, "assume yes to all confirmation prompts")
	pf.BoolVar(&flagNoInput, "no-input", false, "never prompt; use safe defaults (for scripts)")
	pf.BoolVar(&flagNoColor, "no-color", false, "disable colored output (also honors NO_COLOR)")
	_ = rootCmd.RegisterFlagCompletionFunc("scope", completeScope)

	// Validate global flags once and propagate UI settings before any command runs.
	rootCmd.PersistentPreRunE = func(_ *cobra.Command, _ []string) error {
		if !config.Scope(flagScope).Valid() {
			return fmt.Errorf("invalid --scope %q (use \"user\" or \"system\")", flagScope)
		}
		ui.SetAssumeYes(flagYes)
		ui.SetNoInput(flagNoInput)
		ui.SetNoColor(flagNoColor)
		return nil
	}
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
		return config.Config{}, fmt.Errorf("no config at %s — run `vigil config init` first", p)
	}
	return c, err
}
