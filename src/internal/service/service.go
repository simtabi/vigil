// Package service installs, controls and runs vigil as a background
// service. It uses kardianos/service for launchd, systemd and Windows services,
// but on Windows the synthetic-input engine must run in the interactive user
// session, so it is installed as a logon Scheduled Task instead of a session-0
// service.
package service

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"time"

	"github.com/kardianos/service"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/engine"
)

// serviceName is the OS-level service/task identifier.
const serviceName = "vigil"

// Params describes the target install.
type Params struct {
	Scope      config.Scope
	ConfigPath string
	UsesInput  bool // engine includes the synthetic-input strategy
}

// useWindowsTask reports whether this combination must be installed as a Windows
// logon Scheduled Task rather than a session-0 service.
func (p Params) useWindowsTask() bool {
	return runtime.GOOS == "windows" && p.UsesInput
}

// Validate rejects combinations that cannot inject input. A system/daemon scope
// with the input engine runs outside the GUI session and would silently fail.
func (p Params) Validate() error {
	if p.UsesInput && p.Scope == config.ScopeSystem {
		return fmt.Errorf("the input engine requires a GUI session: install with --scope user, " +
			"or switch to the graph engine for a system-wide service")
	}
	return nil
}

func svcConfig(p Params) (*service.Config, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, err
	}
	cfg := &service.Config{
		Name:        serviceName,
		DisplayName: "Vigil",
		Description: "Keeps Microsoft Teams active on a configurable schedule.",
		Executable:  exe,
		Arguments:   []string{"run", "--scope", string(p.Scope), "--config", p.ConfigPath},
		Option:      service.KeyValue{},
	}
	if p.Scope == config.ScopeUser {
		cfg.Option["UserService"] = true
	}
	return cfg, nil
}

type program struct {
	eng    *engine.Engine
	cancel context.CancelFunc
	done   chan struct{}
}

func (pr *program) Start(_ service.Service) error {
	ctx, cancel := context.WithCancel(context.Background())
	pr.cancel = cancel
	pr.done = make(chan struct{})
	go func() {
		defer close(pr.done)
		if err := pr.eng.Run(ctx); err != nil {
			fmt.Fprintln(os.Stderr, "engine error:", err)
		}
	}()
	return nil
}

func (pr *program) Stop(_ service.Service) error {
	if pr.cancel != nil {
		pr.cancel()
	}
	select {
	case <-pr.done:
	case <-time.After(12 * time.Second):
	}
	return nil
}

// Run executes the engine under the platform service supervisor (or foreground
// when launched interactively / from a scheduled task).
func Run(p Params, eng *engine.Engine) error {
	cfg, err := svcConfig(p)
	if err != nil {
		return err
	}
	prg := &program{eng: eng}
	svc, err := service.New(prg, cfg)
	if err != nil {
		return err
	}
	return svc.Run()
}

// Install installs the service/task.
func Install(p Params) (postInstallNote string, err error) {
	if err := p.Validate(); err != nil {
		return "", err
	}
	if p.useWindowsTask() {
		return installWindowsTask(p)
	}
	svc, err := newControlService(p)
	if err != nil {
		return "", err
	}
	if err := svc.Install(); err != nil {
		return "", err
	}
	return postInstallHint(p), svc.Start()
}

// Uninstall stops and removes the service/task.
func Uninstall(p Params) error {
	if p.useWindowsTask() {
		return uninstallWindowsTask(p)
	}
	svc, err := newControlService(p)
	if err != nil {
		return err
	}
	_ = svc.Stop()
	return svc.Uninstall()
}

// Start, Stop, Restart control an installed service/task.
func Start(p Params) error   { return control(p, "start") }
func Stop(p Params) error    { return control(p, "stop") }
func Restart(p Params) error { return control(p, "restart") }

// StatusString returns a human-readable run state.
func StatusString(p Params) (string, error) {
	if p.useWindowsTask() {
		return windowsTaskStatus(p)
	}
	svc, err := newControlService(p)
	if err != nil {
		return "", err
	}
	st, err := svc.Status()
	if err != nil {
		return "", err
	}
	switch st {
	case service.StatusRunning:
		return "running", nil
	case service.StatusStopped:
		return "stopped", nil
	default:
		return "unknown", nil
	}
}

func control(p Params, action string) error {
	if p.useWindowsTask() {
		return controlWindowsTask(p, action)
	}
	svc, err := newControlService(p)
	if err != nil {
		return err
	}
	return service.Control(svc, action)
}

func newControlService(p Params) (service.Service, error) {
	cfg, err := svcConfig(p)
	if err != nil {
		return nil, err
	}
	return service.New(&program{}, cfg)
}

func postInstallHint(p Params) string {
	switch {
	case runtime.GOOS == "linux" && p.Scope == config.ScopeUser:
		return "Tip: run `loginctl enable-linger $USER` so the service keeps running when you're logged out."
	case runtime.GOOS == "darwin" && p.UsesInput:
		return "Tip: grant Accessibility permission to the vigil binary in System Settings → Privacy & Security → Accessibility, then run `vigil doctor`."
	default:
		return ""
	}
}
