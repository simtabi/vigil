package cli

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/control"
	"github.com/simtabi/vigil/internal/netcheck"
	"github.com/simtabi/vigil/internal/schedule"
	"github.com/simtabi/vigil/internal/service"
	"github.com/spf13/cobra"
)

var (
	flagFor   string
	flagUntil string
)

var onCmd = &cobra.Command{
	Use:   "on",
	Short: "Force active now (optionally for a duration or until a time)",
	RunE:  func(_ *cobra.Command, _ []string) error { return setOverride(schedule.OverrideOn) },
}

var offCmd = &cobra.Command{
	Use:   "off",
	Short: "Force inactive now (optionally for a duration or until a time)",
	RunE:  func(_ *cobra.Command, _ []string) error { return setOverride(schedule.OverrideOff) },
}

var resumeCmd = &cobra.Command{
	Use:   "resume",
	Short: "Clear any manual override and resume following the schedule",
	RunE: func(_ *cobra.Command, _ []string) error {
		rt, err := runtimeDir()
		if err != nil {
			return err
		}
		if err := os.Remove(control.OverridePath(rt)); err != nil && !os.IsNotExist(err) {
			return err
		}
		ui.Success("override cleared; following schedule")
		return nil
	},
}

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the daemon and service status",
	RunE:  func(_ *cobra.Command, _ []string) error { return showStatus() },
}

func setOverride(mode schedule.OverrideMode) error {
	rt, err := runtimeDir()
	if err != nil {
		return err
	}
	now := time.Now()
	ov := schedule.Override{Mode: mode, SetAt: now}

	switch {
	case flagFor != "" && flagUntil != "":
		return fmt.Errorf("use only one of --for or --until")
	case flagFor != "":
		d, err := time.ParseDuration(flagFor)
		if err != nil {
			return fmt.Errorf("--for: %w (e.g. 30m, 2h, 1h30m)", err)
		}
		if d <= 0 {
			return fmt.Errorf("--for must be a positive duration, got %q", flagFor)
		}
		u := now.Add(d)
		ov.Until = &u
	case flagUntil != "":
		u, err := parseUntil(now, flagUntil)
		if err != nil {
			return err
		}
		ov.Until = &u
	}

	if err := schedule.SaveOverride(control.OverridePath(rt), ov); err != nil {
		return err
	}
	if ov.Until != nil {
		ui.Success("override %s until %s", mode, ov.Until.Format("Mon 15:04"))
	} else {
		ui.Success("override %s (indefinite; `vigil resume` to clear)", mode)
	}
	return nil
}

// parseUntil interprets "HH:MM" as the next occurrence of that local time.
func parseUntil(now time.Time, s string) (time.Time, error) {
	c, err := config.ParseClock(s)
	if err != nil {
		return time.Time{}, fmt.Errorf("--until: %w", err)
	}
	u := time.Date(now.Year(), now.Month(), now.Day(), c.Minutes()/60, c.Minutes()%60, 0, 0, now.Location())
	if !u.After(now) {
		u = u.Add(24 * time.Hour)
	}
	return u, nil
}

func showStatus() error {
	rt, err := runtimeDir()
	if err != nil {
		return err
	}
	st, statusErr := control.ReadStatus(rt)

	svcState := "unknown"
	if cfg, err := loadConfig(); err == nil {
		cfgPath, _ := configPath()
		if s, err := service.StatusString(service.Params{
			Scope: scope(), ConfigPath: cfgPath, UsesInput: cfg.UsesInput(),
		}); err == nil {
			svcState = s
		}
	}

	netCtx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	online := netcheck.Online(netCtx)
	cancel()

	if flagJSON {
		return printJSON(map[string]any{"service": svcState, "online": online, "daemon": st, "daemon_error": errString(statusErr)})
	}

	fmt.Printf("Service:   %s\n", svcState)
	fmt.Printf("Internet:  %s\n", onlineWord(online))
	if statusErr != nil {
		fmt.Printf("Daemon:    %s\n", statusErr)
		return nil
	}
	fmt.Printf("Engine:    %s\n", st.Engine)
	fmt.Printf("Active:    %v\n", st.DesiredActive)
	if st.OverrideMode != "" {
		line := "Override:  " + st.OverrideMode
		if st.OverrideUntil != nil {
			line += " until " + st.OverrideUntil.Format("Mon 15:04")
		}
		fmt.Println(line)
	}
	if st.NextChange != nil {
		fmt.Printf("Next:      %s at %s\n", activeWord(st.NextActive), st.NextChange.Format("Mon 15:04"))
	}
	if st.LastTick != nil {
		fmt.Printf("Last tick: %s\n", st.LastTick.Format("15:04:05"))
	}
	if st.LastError != "" {
		fmt.Printf("Last err:  %s\n", st.LastError)
	}
	return nil
}

func activeWord(active bool) string {
	if active {
		return "activate"
	}
	return "deactivate"
}

func onlineWord(online bool) string {
	if online {
		return "connected"
	}
	return "offline"
}

func errString(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func init() {
	for _, c := range []*cobra.Command{onCmd, offCmd} {
		c.Flags().StringVar(&flagFor, "for", "", "duration, e.g. 2h30m")
		c.Flags().StringVar(&flagUntil, "until", "", "local time HH:MM")
	}
	rootCmd.AddCommand(onCmd, offCmd, resumeCmd, statusCmd)
}
