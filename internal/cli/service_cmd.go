package cli

import (
	"fmt"

	"github.com/simtabi/ms-teams-activity/internal/service"
	"github.com/spf13/cobra"
)

// serviceParams builds install/control params from the current config + scope.
func serviceParams() (service.Params, error) {
	cfg, err := loadConfig()
	if err != nil {
		return service.Params{}, err
	}
	cfgPath, err := configPath()
	if err != nil {
		return service.Params{}, err
	}
	return service.Params{Scope: scope(), ConfigPath: cfgPath, UsesInput: cfg.UsesInput()}, nil
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and start the background service (auto-selects the right mechanism)",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := serviceParams()
		if err != nil {
			return err
		}
		note, err := service.Install(p)
		if err != nil {
			return err
		}
		fmt.Println("installed and started")
		if note != "" {
			fmt.Println(note)
		}
		return nil
	},
}

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Stop and remove the background service",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := serviceParams()
		if err != nil {
			return err
		}
		if err := service.Uninstall(p); err != nil {
			return err
		}
		fmt.Println("uninstalled")
		return nil
	},
}

func simpleServiceCmd(use, short string, fn func(service.Params) error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(_ *cobra.Command, _ []string) error {
			p, err := serviceParams()
			if err != nil {
				return err
			}
			if err := fn(p); err != nil {
				return err
			}
			fmt.Println(use + ": ok")
			return nil
		},
	}
}

func init() {
	rootCmd.AddCommand(
		installCmd,
		uninstallCmd,
		simpleServiceCmd("start", "Start the installed service", service.Start),
		simpleServiceCmd("stop", "Stop the installed service", service.Stop),
		simpleServiceCmd("restart", "Restart the installed service", service.Restart),
	)
}
