package cli

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
)

// Build metadata, overridable via -ldflags "-X .../cmd.version=...".
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	RunE: func(_ *cobra.Command, _ []string) error {
		if flagJSON {
			return printJSON(map[string]string{
				"version": version, "commit": commit, "date": date,
				"go": runtime.Version(), "platform": runtime.GOOS + "/" + runtime.GOARCH,
			})
		}
		fmt.Printf("mta %s (commit %s, built %s, %s %s/%s)\n",
			version, commit, date, runtime.Version(), runtime.GOOS, runtime.GOARCH)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
