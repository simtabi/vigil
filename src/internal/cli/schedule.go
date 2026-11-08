package cli

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/spf13/cobra"
)

var (
	flagDays  string
	flagStart string
	flagEnd   string
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "Manage schedule windows from the CLI",
}

var scheduleListCmd = &cobra.Command{
	Use:   "list",
	Short: "List schedule windows",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := loadConfig()
		if err != nil {
			return err
		}
		if flagJSON {
			return printJSON(c.Schedule)
		}
		fmt.Printf("enabled=%v always=%v tz=%s\n", c.Schedule.Enabled, c.Schedule.Always, c.Timezone)
		if len(c.Schedule.Windows) == 0 {
			fmt.Println("(no windows)")
			return nil
		}
		for i, w := range c.Schedule.Windows {
			fmt.Printf("  [%d] %s  %s–%s\n", i, strings.Join(w.Days, ","), w.Start, w.End)
		}
		return nil
	},
}

var scheduleAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a window, e.g. --days Mon,Tue,Wed,Thu,Fri --start 08:00 --end 17:00",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		c, err := loadConfig()
		if err != nil {
			return err
		}
		days := splitDays(flagDays)
		if len(days) == 0 {
			return fmt.Errorf("--days is required (comma-separated, e.g. Mon,Tue)")
		}
		w := config.Window{Days: days, Start: flagStart, End: flagEnd}
		c.Schedule.Windows = append(c.Schedule.Windows, w)
		if err := c.Validate(); err != nil {
			return err
		}
		if err := c.Save(p); err != nil {
			return err
		}
		ui.Success("added window: %s %s–%s", strings.Join(days, ","), flagStart, flagEnd)
		return nil
	},
}

var scheduleRemoveCmd = &cobra.Command{
	Use:   "remove <index>",
	Short: "Remove a window by its list index",
	Args:  cobra.ExactArgs(1),
	RunE: func(_ *cobra.Command, args []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		c, err := loadConfig()
		if err != nil {
			return err
		}
		idx, err := strconv.Atoi(args[0])
		if err != nil || idx < 0 || idx >= len(c.Schedule.Windows) {
			return fmt.Errorf("invalid index %q (have %d windows)", args[0], len(c.Schedule.Windows))
		}
		c.Schedule.Windows = append(c.Schedule.Windows[:idx], c.Schedule.Windows[idx+1:]...)
		if err := c.Save(p); err != nil {
			return err
		}
		ui.Success("removed window %d", idx)
		return nil
	},
}

var scheduleClearCmd = &cobra.Command{
	Use:   "clear",
	Short: "Remove all windows",
	RunE: func(_ *cobra.Command, _ []string) error {
		p, err := configPath()
		if err != nil {
			return err
		}
		c, err := loadConfig()
		if err != nil {
			return err
		}
		if len(c.Schedule.Windows) > 0 && !ui.Confirm(fmt.Sprintf("Remove all %d schedule window(s)?", len(c.Schedule.Windows)), false) {
			ui.Info("cancelled")
			return nil
		}
		c.Schedule.Windows = nil
		if err := c.Save(p); err != nil {
			return err
		}
		ui.Success("cleared all windows")
		return nil
	},
}

func splitDays(s string) []string {
	var out []string
	for _, d := range strings.Split(s, ",") {
		d = strings.TrimSpace(d)
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

func init() {
	scheduleAddCmd.Flags().StringVar(&flagDays, "days", "", "comma-separated days (Mon..Sun)")
	scheduleAddCmd.Flags().StringVar(&flagStart, "start", "09:00", "start time HH:MM")
	scheduleAddCmd.Flags().StringVar(&flagEnd, "end", "17:00", "end time HH:MM")
	scheduleCmd.AddCommand(scheduleListCmd, scheduleAddCmd, scheduleRemoveCmd, scheduleClearCmd)
	rootCmd.AddCommand(scheduleCmd)
}
