package cli

import (
	"encoding/json"
	"fmt"
	"os"
)

// errAlreadyRunning reports that another daemon holds the runtime lock.
func errAlreadyRunning(rt string) error {
	return fmt.Errorf("another vigil daemon is already running for %s (use `vigil stop` or a different --scope)", rt)
}

// printJSON writes v as indented JSON to stdout.
func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}
