package cli

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/simtabi/ms-teams-activity/internal/config"
	"github.com/simtabi/ms-teams-activity/internal/control"
	"github.com/simtabi/ms-teams-activity/internal/engine"
	"github.com/simtabi/ms-teams-activity/internal/service"
	"github.com/spf13/cobra"
	"gopkg.in/natefinch/lumberjack.v2"
)

var flagForeground bool

var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Run the activity daemon (used by the service; can be run manually)",
	RunE: func(_ *cobra.Command, _ []string) error {
		cfgPath, err := configPath()
		if err != nil {
			return err
		}
		cfg, err := loadConfig()
		if err != nil {
			return err
		}
		rt, err := runtimeDir()
		if err != nil {
			return err
		}
		tokenPath, err := config.TokenPath(scope())
		if err != nil {
			return err
		}

		lock, ok, err := control.AcquireLock(rt)
		if err != nil {
			return err
		}
		if !ok {
			return errAlreadyRunning(rt)
		}
		defer func() { _ = lock.Release() }()

		log := newLogger(cfg, rt)
		eng := engine.New(scope(), cfgPath, rt, tokenPath, log)

		params := service.Params{Scope: scope(), ConfigPath: cfgPath, UsesInput: cfg.UsesInput()}
		return service.Run(params, eng)
	},
}

func init() {
	runCmd.Flags().BoolVar(&flagForeground, "foreground", false, "(reserved) force foreground execution")
	rootCmd.AddCommand(runCmd)
}

// newLogger builds a slog logger that writes a rotating log file in the runtime
// dir and also mirrors to stderr (useful when run manually / by systemd-journal).
func newLogger(cfg config.Config, rt string) *slog.Logger {
	rotator := &lumberjack.Logger{
		Filename:   control.LogPath(rt),
		MaxSize:    max(cfg.Log.MaxSizeMB, 1),
		MaxBackups: max(cfg.Log.MaxBackups, 0),
		Compress:   true,
	}
	w := io.MultiWriter(os.Stderr, rotator)
	var lvl slog.Level
	switch strings.ToLower(cfg.Log.Level) {
	case "debug":
		lvl = slog.LevelDebug
	case "warn":
		lvl = slog.LevelWarn
	case "error":
		lvl = slog.LevelError
	default:
		lvl = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{Level: lvl}))
}
