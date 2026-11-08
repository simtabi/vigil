package cli

import (
	"context"
	"fmt"

	"github.com/simtabi/vigil/internal/cli/ui"
	"github.com/simtabi/vigil/internal/config"
	"github.com/simtabi/vigil/internal/graph"
	"github.com/spf13/cobra"
)

func graphClient() (*graph.Client, error) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Graph.ClientID == "" {
		return nil, fmt.Errorf("graph.client_id is empty — set it in config (register an Entra public-client app)")
	}
	tokenPath, err := config.TokenPath(scope())
	if err != nil {
		return nil, err
	}
	return graph.New(cfg.Graph.TenantID, cfg.Graph.ClientID, tokenPath)
}

var authCmd = &cobra.Command{
	Use:   "auth",
	Short: "Manage Microsoft Graph authentication",
}

var authLoginCmd = &cobra.Command{
	Use:   "login",
	Short: "Sign in via the device-code flow (requires admin-consented Presence.ReadWrite)",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := graphClient()
		if err != nil {
			return err
		}
		if err := c.Login(context.Background(), func(msg string) { fmt.Println(msg) }); err != nil {
			return err
		}
		ui.Success("signed in; token cached")
		return nil
	},
}

var authStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show the signed-in Graph account",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := graphClient()
		if err != nil {
			return err
		}
		acct, err := c.Account(context.Background())
		if err != nil {
			return err
		}
		if flagJSON {
			return printJSON(map[string]any{"account": acct, "signed_in": acct != ""})
		}
		if acct == "" {
			fmt.Println("not signed in (run `vigil auth login`)")
			return nil
		}
		fmt.Printf("signed in as %s\n", acct)
		return nil
	},
}

var authLogoutCmd = &cobra.Command{
	Use:   "logout",
	Short: "Remove cached Graph credentials",
	RunE: func(_ *cobra.Command, _ []string) error {
		c, err := graphClient()
		if err != nil {
			return err
		}
		if !ui.Confirm("Sign out and remove cached credentials?", false) {
			ui.Info("cancelled")
			return nil
		}
		if err := c.Logout(context.Background()); err != nil {
			return err
		}
		ui.Success("signed out")
		return nil
	},
}

func init() {
	authCmd.AddCommand(authLoginCmd, authStatusCmd, authLogoutCmd)
	rootCmd.AddCommand(authCmd)
}
