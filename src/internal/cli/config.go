package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/spf13/cobra"
)

var flagForce bool

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage the JSON configuration",
}

var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Write a default config.json for the current scope",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(p); err == nil && !flagForce {
			if !ui.Confirm(fmt.Sprintf("%s exists. Overwrite with defaults?", p), false) {
				ui.Info("cancelled")
				return nil
			}
		}
		if err := config.Default().Save(p); err != nil {
			return err
		}
		ui.Success("wrote default config to %s", p)
		return nil
	},
}

var configPathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the effective config path",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		fmt.Println(p)
		return nil
	},
}

var configValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate the config and report problems",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		if flagJSON {
			return printJSON(map[string]any{"valid": true, "engine": c.Engine})
		}
		ui.Success("config is valid")
		return nil
	},
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the effective config",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		return printJSON(c)
	},
}

var configEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Open the config in $EDITOR",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		if _, err := os.Stat(p); os.IsNotExist(err) {
			return fmt.Errorf("no config at %s — run `vigil config init` first", p)
		}
		editor := os.Getenv("EDITOR")
		if editor == "" {
			if runtime.GOOS == "windows" {
				editor = "notepad"
			} else {
				editor = "vi"
			}
		}
		c := exec.Command(editor, p)
		c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
		if err := c.Run(); err != nil {
			return err
		}
		// Validate after editing so mistakes surface immediately.
		if _, err := config.Load(p); err != nil {
			return fmt.Errorf("saved, but config is now invalid: %w", err)
		}
		ui.Success("config saved and valid")
		return nil
	},
}

var configGetCmd = &cobra.Command{
	Use:   "get <key>",
	Short: "Print a single config value (dotted key, e.g. input.interval_seconds)",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		v, err := c.GetField(args[0])
		if err != nil {
			return err
		}
		fmt.Println(v)
		return nil
	},
}

var configSetCmd = &cobra.Command{
	Use:   "set <key> <value>",
	Short: "Set a config value (validated) and save",
	Args:  cobra.ExactArgs(2),
	RunE: func(_ *cobra.Command, args []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		c, err := loadConfig()
		if err != nil {
			return err
		}
		if err := c.SetField(args[0], args[1]); err != nil {
			return err
		}
		if err := c.Save(p); err != nil {
			return err
		}
		ui.Success("%s = %s", args[0], args[1])
		return nil
	},
}

var configKeysCmd = &cobra.Command{
	Use:   "keys",
	Short: "List settable config keys",
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagJSON {
			return printJSON(config.SettableKeys())
		}
		for _, k := range config.SettableKeys() {
			fmt.Println(k)
		}
		return nil
	},
}

var configWizardCmd = &cobra.Command{
	Use:   "wizard",
	Short: "Guided interactive setup",
	RunE:  func(_ *cobra.Command, _ []string) error { return runWizard() },
}

func init() {
	configInitCmd.Flags().BoolVar(&flagForce, "force", false, "overwrite an existing config")
	configCmd.AddCommand(
		configInitCmd, configPathCmd, configValidateCmd, configShowCmd, configEditCmd,
		configGetCmd, configSetCmd, configKeysCmd, configWizardCmd,
	)
	rootCmd.AddCommand(configCmd)
}
