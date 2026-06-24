//go:build windows

package service

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// On Windows the input engine must run in the interactive desktop session, so we
// register a per-user logon Scheduled Task (RL LIMITED) instead of a session-0
// service.

func taskCommand(p Params) (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf(`"%s" run --scope %s --config "%s"`, exe, p.Scope, p.ConfigPath), nil
}

func installWindowsTask(p Params) (string, error) {
	tr, err := taskCommand(p)
	if err != nil {
		return "", err
	}
	cmd := exec.Command("schtasks", "/Create", "/TN", serviceName, "/TR", tr,
		"/SC", "ONLOGON", "/RL", "LIMITED", "/F")
	if out, err := cmd.CombinedOutput(); err != nil {
		return "", fmt.Errorf("schtasks create failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	if out, err := exec.Command("schtasks", "/Run", "/TN", serviceName).CombinedOutput(); err != nil {
		return "", fmt.Errorf("schtasks run failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return "Installed as a logon Scheduled Task; it starts automatically each time you sign in.", nil
}

func uninstallWindowsTask(_ Params) error {
	if out, err := exec.Command("schtasks", "/End", "/TN", serviceName).CombinedOutput(); err != nil {
		_ = out // ignore: task may not be running
	}
	if out, err := exec.Command("schtasks", "/Delete", "/TN", serviceName, "/F").CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks delete failed: %v: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

func controlWindowsTask(_ Params, action string) error {
	var args []string
	switch action {
	case "start":
		args = []string{"/Run", "/TN", serviceName}
	case "stop":
		args = []string{"/End", "/TN", serviceName}
	case "restart":
		_ = exec.Command("schtasks", "/End", "/TN", serviceName).Run()
		args = []string{"/Run", "/TN", serviceName}
	default:
		return fmt.Errorf("unknown action %q", action)
	}
	if out, err := exec.Command("schtasks", args...).CombinedOutput(); err != nil {
		return fmt.Errorf("schtasks %s failed: %v: %s", action, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func windowsTaskStatus(_ Params) (string, error) {
	out, err := exec.Command("schtasks", "/Query", "/TN", serviceName, "/FO", "LIST").CombinedOutput()
	if err != nil {
		return "not installed", nil
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "Status:") {
			st := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(line), "Status:"))
			return strings.ToLower(st), nil
		}
	}
	return "unknown", nil
}
