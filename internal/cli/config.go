package cli

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"

	"github.com/simtabi/ms-teams-activity/internal/config"
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
			return fmt.Errorf("%s already exists (use --force to overwrite)", p)
		}
		if err := config.Default().Save(p); err != nil {
			return err
		}
		fmt.Printf("Wrote default config to %s\n", p)
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
		fmt.Println("config is valid")
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
			return fmt.Errorf("no config at %s — run `mta config init` first", p)
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
		fmt.Println("config saved and valid")
		return nil
	},
}

func init() {
	configInitCmd.Flags().BoolVar(&flagForce, "force", false, "overwrite an existing config")
	configCmd.AddCommand(configInitCmd, configPathCmd, configValidateCmd, configShowCmd, configEditCmd)
	rootCmd.AddCommand(configCmd)
}
