package cmd

import (
	"github.com/simtabi/ms-teams-activity/tui"
	"github.com/spf13/cobra"
)

var tuiCmd = &cobra.Command{
	Use:   "tui",
	Short: "Open the interactive dashboard",
	RunE:  func(_ *cobra.Command, _ []string) error { return runTUI() },
}

func runTUI() error {
	cfgPath, err := configPath()
	if err != nil {
		return err
	}
	rt, err := runtimeDir()
	if err != nil {
		return err
	}
	return tui.Run(tui.Options{Scope: scope(), ConfigPath: cfgPath, RuntimeDir: rt})
}

func init() {
	rootCmd.AddCommand(tuiCmd)
}
