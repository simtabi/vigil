package cli

import (
	"github.com/simtabi/vigil/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive dashboard",
	RunE:  func(_ *cobra.Command, _ []string) error { return runTUI() },
}

func tuiOptions() (tui.Options, error) {
	cfgPath, err := configPath()
	if err != nil {
		return tui.Options{}, err
	}
	rt, err := runtimeDir()
	if err != nil {
		return tui.Options{}, err
	}
	return tui.Options{Scope: scope(), ConfigPath: cfgPath, RuntimeDir: rt, Version: version}, nil
}

func runTUI() error {
	opts, err := tuiOptions()
	if err != nil {
		return err
	}
	return tui.Run(opts)
}

func runWizard() error {
	opts, err := tuiOptions()
	if err != nil {
		return err
	}
	return tui.RunWizard(opts)
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
