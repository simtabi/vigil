package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/simtabi/ms-teams-activity/internal/activity"
	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/spf13/cobra"
)

type checkLevel string

const (
	levelOK   checkLevel = "OK"
	levelWarn checkLevel = "WARN"
	levelFail checkLevel = "FAIL"
)

type check struct {
	Level  checkLevel `json:"level"`
	Name   string     `json:"name"`
	Detail string     `json:"detail"`
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Diagnose platform capabilities, permissions and configuration",
	RunE: func(_ *cobra.Command, _ []string) error {
		var checks []check
		add := func(l checkLevel, n, d string) { checks = append(checks, check{l, n, d}) }

		add(levelOK, "platform", fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH))

		cfg, cfgErr := loadConfig()
		if cfgErr != nil {
			add(levelWarn, "config", cfgErr.Error())
		} else {
			add(levelOK, "config", "engine="+string(cfg.Engine))
		}

		checkInput(cfg, cfgErr, add)
		checkGraph(cfg, cfgErr, add)
		checkScope(cfg, cfgErr, add)

		if flagJSON {
			return printJSON(checks)
		}
		worst := levelOK
		for _, c := range checks {
			fmt.Printf("[%-4s] %-14s %s\n", c.Level, c.Name, c.Detail)
			if c.Level == levelFail || (c.Level == levelWarn && worst == levelOK) {
				worst = c.Level
			}
		}
		fmt.Printf("\nsummary: %s\n", worst)
		return nil
	},
}

func checkInput(cfg config.Config, cfgErr error, add func(checkLevel, string, string)) {
	if cfgErr == nil && !cfg.UsesInput() {
		return
	}
	if !activity.InputSupported() {
		add(levelFail, "input-engine", "no synthetic-input backend on this OS; use the graph engine")
		return
	}
	switch runtime.GOOS {
	case "darwin":
		if activity.AccessibilityTrusted() {
			add(levelOK, "accessibility", "process is trusted for Accessibility")
		} else {
			add(levelFail, "accessibility", "NOT trusted — grant Accessibility to mta in System Settings → Privacy & Security → Accessibility")
		}
		checkMacScreensaver(cfg, add)
	case "linux":
		checkLinuxUinput(add)
	case "windows":
		add(levelOK, "input-engine", "SendInput available")
	}
}

func checkMacScreensaver(cfg config.Config, add func(checkLevel, string, string)) {
	out, err := exec.Command("defaults", "-currentHost", "read", "com.apple.screensaver", "idleTime").Output()
	if err != nil {
		add(levelWarn, "auto-lock", "could not detect screensaver idle time; ensure it exceeds the interval or is disabled")
		return
	}
	secs, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return
	}
	if secs == 0 {
		add(levelOK, "auto-lock", "screensaver disabled")
		return
	}
	if secs <= cfg.Input.IntervalSeconds {
		add(levelWarn, "auto-lock", fmt.Sprintf("screensaver starts after %ds but interval is %ds — synthetic input cannot reset the hardware idle timer, so the screen may still lock (Teams → Away). Increase the timeout or disable it.", secs, cfg.Input.IntervalSeconds))
		return
	}
	add(levelOK, "auto-lock", fmt.Sprintf("screensaver after %ds > interval %ds", secs, cfg.Input.IntervalSeconds))
}

func checkLinuxUinput(add func(checkLevel, string, string)) {
	if _, err := os.Stat("/dev/uinput"); err != nil {
		add(levelFail, "uinput", "/dev/uinput not found — load the uinput module (modprobe uinput)")
		return
	}
	f, err := os.OpenFile("/dev/uinput", os.O_WRONLY, 0)
	if err != nil {
		add(levelFail, "uinput", "/dev/uinput not writable — add your user to a group with access (see docs) and add a udev rule")
		return
	}
	_ = f.Close()
	add(levelOK, "uinput", "/dev/uinput present and writable")
}

func checkGraph(cfg config.Config, cfgErr error, add func(checkLevel, string, string)) {
	if cfgErr != nil || !cfg.UsesGraph() {
		return
	}
	c, err := graphClient()
	if err != nil {
		add(levelFail, "graph", err.Error())
		return
	}
	acct, err := c.Account(context.Background())
	if err != nil {
		add(levelWarn, "graph", "could not read account: "+err.Error())
		return
	}
	if acct == "" {
		add(levelWarn, "graph", "not signed in — run `mta auth login` (needs admin-consented Presence.ReadWrite)")
		return
	}
	add(levelOK, "graph", "signed in as "+acct)
}

func checkScope(cfg config.Config, cfgErr error, add func(checkLevel, string, string)) {
	if cfgErr == nil && cfg.UsesInput() && scope() == config.ScopeSystem {
		add(levelFail, "scope", "input engine with --scope system cannot inject into the GUI session; use --scope user")
	}
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
