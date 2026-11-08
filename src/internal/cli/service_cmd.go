package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/service"
	"github.com/spf13/cobra"
)

var flagInstallInit bool

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

// ensureConfig writes a default config at path when none exists yet. It returns
// whether it created the file.
func ensureConfig(path string) (bool, error) {
	if _, err := os.Stat(path); err == nil {
		return false, nil
	} else if !os.IsNotExist(err) {
		return false, err
	}
	if err := config.Default().Save(path); err != nil {
		return false, err
	}
	return true, nil
}

func alreadyInstalled(err error) bool {
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "already exists") || strings.Contains(e, "already installed")
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install and start the background service (auto-selects the right mechanism)",
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagInstallInit {
			p, err := configPath()
			if err != nil {
				return err
			}
			created, err := ensureConfig(p)
			if err != nil {
				return err
			}
			if created {
				fmt.Println("wrote default config:", p)
			}
		}
		p, err := serviceParams()
		if err != nil {
			return err
		}
		var note string
		err = ui.Spin("Installing service", func() error {
			var e error
			note, e = service.Install(p)
			return e
		})
		if err != nil {
			if alreadyInstalled(err) {
				ui.Info("service already installed; restarting")
				if e := ui.Spin("Restarting service", func() error { return service.Restart(p) }); e != nil {
					return e
				}
				ui.Success("restarted")
				return nil
			}
			return err
		}
		ui.Success("installed and started")
		if note != "" {
			ui.Info("%s", note)
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
		if !ui.Confirm("Remove the background service?", false) {
			ui.Info("cancelled")
			return nil
		}
		if err := ui.Spin("Removing service", func() error { return service.Uninstall(p) }); err != nil {
			return err
		}
		ui.Success("uninstalled")
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
				return fmt.Errorf("%s failed: %w (is the service installed? run `vigil install`)", use, err)
			}
			ui.Success("%s: ok", use)
			return nil
		},
	}
}

func init() {
	installCmd.Flags().BoolVar(&flagInstallInit, "init", false, "write a default config first if none exists (turnkey)")
	rootCmd.AddCommand(
		installCmd,
		uninstallCmd,
		simpleServiceCmd("start", "Start the installed service", service.Start),
		simpleServiceCmd("stop", "Stop the installed service", service.Stop),
		simpleServiceCmd("restart", "Restart the installed service", service.Restart),
	)
}
