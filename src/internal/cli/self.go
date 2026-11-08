package cli

import (
	"context"
	"fmt"
	"os"
	"runtime"

	"github.com/simtabi/vigil/src/internal/cli/ui"
	"github.com/simtabi/vigil/src/internal/config"
	"github.com/simtabi/vigil/src/internal/selfupdate"
	"github.com/simtabi/vigil/src/internal/service"
	"github.com/spf13/cobra"
)

var (
	flagCheck        bool
	flagUpgradeForce bool
	flagPurge        bool
)

var upgradeCmd = &cobra.Command{
	Use:     "upgrade",
	Short:   "Update vigil to the latest release (alias of `self update`)",
	Example: "  vigil upgrade --check\n  vigil upgrade --yes\n  vigil upgrade --force",
	RunE:    func(_ *cobra.Command, _ []string) error { return doUpgrade() },
}

var selfCmd = &cobra.Command{
	Use:   "self",
	Short: "Manage the vigil binary itself (update/install/uninstall)",
}

var selfUpdateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update vigil to the latest release",
	RunE:  func(_ *cobra.Command, _ []string) error { return doUpgrade() },
}

var selfInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Copy this binary to a standard location on PATH",
	RunE:  func(_ *cobra.Command, _ []string) error { return doSelfInstall() },
}

var selfUninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the service and the vigil binary (--purge also deletes config/data)",
	RunE:  func(_ *cobra.Command, _ []string) error { return doSelfUninstall() },
}

func doUpgrade() error {
	ctx := context.Background()
	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	ch := selfupdate.DetectChannel(exe)

	// Always allow a check; only block the apply for dev/package-managed.
	if flagCheck {
		var info selfupdate.Info
		if err := ui.Spin("Checking for updates", func() error {
			var e error
			info, e = selfupdate.Check(ctx, version)
			return e
		}); err != nil {
			return err
		}
		printCheck(info, ch)
		return nil
	}

	if selfupdate.IsDev(version) && !flagUpgradeForce {
		return fmt.Errorf("%w (use a released build, `vigil upgrade --check`, or --force)", selfupdate.ErrDevVersion)
	}
	if !ch.SelfUpdatable() {
		return fmt.Errorf("not self-updating: %s", ch.Advice())
	}

	var info selfupdate.Info
	if err := ui.Spin("Checking for updates", func() error {
		var e error
		info, e = selfupdate.Check(ctx, version)
		return e
	}); err != nil {
		return err
	}
	if info.Latest == "" {
		ui.Info("no releases found")
		return nil
	}
	// The downgrade guard: a release older than this build is refused unless forced.
	install, err := selfupdate.ShouldInstall(version, info.Latest, flagUpgradeForce)
	if err != nil {
		return err
	}
	if !install {
		ui.Info("already up to date (%s)", version)
		return nil
	}
	prompt := fmt.Sprintf("Update %s → %s?", info.Current, info.Latest)
	if info.Ahead {
		prompt = fmt.Sprintf("Replace %s with the older release %s?", displayCurrent(info.Current), info.Latest)
	}
	if !ui.Confirm(prompt, false) {
		ui.Info("cancelled")
		return nil
	}

	// Stop a running service so the binary file can be replaced cleanly.
	params, haveParams := serviceParamsBestEffort()
	wasRunning := false
	if haveParams {
		if st, err := service.StatusString(params); err == nil && st == "running" {
			wasRunning = true
			_ = ui.Spin("Stopping service", func() error { return service.Stop(params) })
		}
	}

	var applied selfupdate.Info
	applyErr := ui.Spin(fmt.Sprintf("Downloading %s", info.Latest), func() error {
		var e error
		applied, e = selfupdate.ApplyWith(ctx, version, flagUpgradeForce)
		return e
	})
	if wasRunning {
		_ = ui.Spin("Restarting service", func() error { return service.Start(params) })
	}
	if applyErr != nil {
		return applyErr
	}
	if !applied.Available {
		ui.Info("already up to date (%s)", version)
		return nil
	}
	ui.Success("updated to %s", applied.Latest)
	tccReminderIfNeeded()
	return nil
}

func printCheck(info selfupdate.Info, ch selfupdate.Channel) {
	if flagJSON {
		_ = printJSON(map[string]any{
			"current": info.Current, "latest": info.Latest,
			"available": info.Available, "ahead": info.Ahead, "channel": ch.String(),
			"self_updatable": ch.SelfUpdatable(),
		})
		return
	}
	switch {
	case info.Latest == "":
		fmt.Println("no releases found")
	case info.Available:
		fmt.Printf("update available: %s → %s\n", info.Current, info.Latest)
		if !ch.SelfUpdatable() {
			fmt.Println(ch.Advice())
		}
	case info.Ahead:
		fmt.Printf("no update: %s is newer than the latest release %s\n", displayCurrent(info.Current), info.Latest)
	default:
		fmt.Printf("up to date (%s)\n", info.Current)
	}
}

// displayCurrent names the running version, including an unversioned build.
func displayCurrent(v string) string {
	if v == "" {
		return "this build"
	}
	return v
}

func doSelfInstall() error {
	src, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	dir, err := binInstallDir(scope())
	if err != nil {
		return err
	}
	dst := dir + string(os.PathSeparator) + binName()
	if err := copyFile(src, dst, 0o755); err != nil {
		return err
	}
	ui.Success("installed: %s", dst)
	ui.Info("ensure %s is on your PATH, then run `vigil config wizard` and `vigil install`.", dir)
	tccReminderIfNeeded()
	return nil
}

func doSelfUninstall() error {
	if !ui.Confirm("Remove the vigil service and binary?", false) {
		ui.Info("cancelled")
		return nil
	}
	// Remove the service/task first (best effort).
	if params, ok := serviceParamsBestEffort(); ok {
		if err := ui.Spin("Removing service", func() error { return service.Uninstall(params) }); err != nil {
			ui.Warn("service uninstall: %v", err)
		}
	}

	if flagPurge {
		if dir, err := config.ConfigDir(scope()); err == nil {
			_ = os.RemoveAll(dir)
		}
		if dir, err := config.RuntimeDir(scope()); err == nil {
			_ = os.RemoveAll(dir)
		}
		ui.Info("removed config and runtime data")
	}

	exe, err := selfupdate.ExecutablePath()
	if err != nil {
		return err
	}
	if runtime.GOOS == "windows" {
		ui.Warn("service removed. Delete the binary manually (it is running): %s", exe)
		return nil
	}
	if err := os.Remove(exe); err != nil {
		ui.Warn("could not remove binary %s: %v", exe, err)
		return nil
	}
	ui.Success("removed binary: %s", exe)
	return nil
}

func init() {
	upgradeCmd.Flags().BoolVar(&flagCheck, "check", false, "only report whether an update is available")
	selfUpdateCmd.Flags().BoolVar(&flagCheck, "check", false, "only report whether an update is available")
	upgradeCmd.Flags().BoolVar(&flagUpgradeForce, "force", false, "install the latest release even if it is older than this build")
	selfUpdateCmd.Flags().BoolVar(&flagUpgradeForce, "force", false, "install the latest release even if it is older than this build")
	selfUninstallCmd.Flags().BoolVar(&flagPurge, "purge", false, "also delete config and runtime data")

	selfCmd.AddCommand(selfUpdateCmd, selfInstallCmd, selfUninstallCmd)
	rootCmd.AddCommand(upgradeCmd, selfCmd)
}
