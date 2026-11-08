package cli

import (
	"fmt"
	"strings"

	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/tz"
	"github.com/spf13/cobra"
)

// valueCandidates returns the completion candidates for a `config set <key>`
// value, or nil (with file completion disabled) for free-form/typed values.
func valueCandidates(key string) ([]string, bool) {
	switch key {
	case "engine":
		return []string{
			string(config.EngineInput),
			string(config.EngineGraph),
			string(config.EngineBoth),
		}, true
	case "input.method":
		return []string{
			string(config.MethodMouse),
			string(config.MethodKey),
			string(config.MethodZen),
		}, true
	case "timezone":
		return tz.Zones, true
	case "log.level":
		return []string{"debug", "info", "warn", "error"}, true
	case "schedule.enabled", "schedule.always", "input.prevent_sleep":
		return []string{"true", "false"}, true
	default:
		return nil, false
	}
}

// completeConfigKey completes the first arg of `config get`/`config set`.
func completeConfigKey(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return config.SettableKeys(), cobra.ShellCompDirectiveNoFileComp
}

// completeConfigSet completes `config set <key> <value>`: keys first, then
// per-key value candidates (timezones are prefix-filtered since the list is huge).
func completeConfigSet(_ *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	switch len(args) {
	case 0:
		return config.SettableKeys(), cobra.ShellCompDirectiveNoFileComp
	case 1:
		vals, ok := valueCandidates(args[0])
		if !ok {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		if args[0] == "timezone" {
			vals = prefixFilter(vals, toComplete)
		}
		return vals, cobra.ShellCompDirectiveNoFileComp
	default:
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
}

// completeScope completes the --scope flag value.
func completeScope(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return []string{"user", "system"}, cobra.ShellCompDirectiveNoFileComp
}

func prefixFilter(items []string, prefix string) []string {
	if prefix == "" {
		return items
	}
	p := strings.ToLower(prefix)
	out := make([]string, 0, 16)
	for _, it := range items {
		if strings.HasPrefix(strings.ToLower(it), p) {
			out = append(out, it)
		}
	}
	return out
}

// completeScheduleIndex completes `schedule remove <index>` with the current
// window indices (annotated with their days/times).
func completeScheduleIndex(_ *cobra.Command, args []string, _ string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	cfg, err := loadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	out := make([]string, 0, len(cfg.Schedule.Windows))
	for i, w := range cfg.Schedule.Windows {
		out = append(out, fmt.Sprintf("%d\t%s %s-%s", i, strings.Join(w.Days, ","), w.Start, w.End))
	}
	return out, cobra.ShellCompDirectiveNoFileComp
}

func init() {
	configGetCmd.ValidArgsFunction = completeConfigKey
	configSetCmd.ValidArgsFunction = completeConfigSet
	scheduleRemoveCmd.ValidArgsFunction = completeScheduleIndex
}
