// Package graph wraps Microsoft Graph authentication (MSAL device-code flow with
// a persistent token cache) and the presence "preferred presence" endpoints.
package graph

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/AzureAD/microsoft-authentication-library-for-go/apps/public"
)

// Scope is the delegated permission required to set presence. Note: this
// permission requires tenant ADMIN consent.
const Scope = "https://graph.microsoft.com/Presence.ReadWrite"

const graphBase = "https://graph.microsoft.com/v1.0"

// ErrLoginRequired is returned when no cached account can produce a token
// silently and an interactive `auth login` is needed.
var ErrLoginRequired = errors.New("graph: login required (run `vigil auth login`)")

// Client authenticates against Entra ID and calls the presence endpoints.
type Client struct {
	pca    public.Client
	scopes []string
	http   *http.Client
}

// New constructs a Client for the given Entra public-client app. tokenPath is
// where the MSAL token cache is persisted.
func New(tenantID, clientID, tokenPath string) (*Client, error) {
	if clientID == "" {
		return nil, errors.New("graph: client_id is empty")
	}
	authority := "https://login.microsoftonline.com/" + tenantID
	pca, err := public.New(clientID,
		public.WithAuthority(authority),
		public.WithCache(newFileCache(tokenPath)),
	)
	if err != nil {
		return nil, fmt.Errorf("graph: init MSAL: %w", err)
	}
	return &Client{
		pca:    pca,
		scopes: []string{Scope},
		http:   &http.Client{Timeout: 20 * time.Second},
	}, nil
}

// Login performs the device-code flow. prompt is called with the user-facing
// instruction message (URL + code). On success the token is cached.
func (c *Client) Login(ctx context.Context, prompt func(message string)) error {
	dc, err := c.pca.AcquireTokenByDeviceCode(ctx, c.scopes)
	if err != nil {
		return fmt.Errorf("graph: start device code: %w", err)
	}
	if prompt != nil {
		prompt(dc.Result.Message)
	}
	if _, err := dc.AuthenticationResult(ctx); err != nil {
		return fmt.Errorf("graph: complete device code: %w", err)
	}
	return nil
}

// Logout removes the cached account(s).
func (c *Client) Logout(ctx context.Context) error {
	accounts, err := c.pca.Accounts(ctx)
	if err != nil {
		return err
	}
	for _, a := range accounts {
		if err := c.pca.RemoveAccount(ctx, a); err != nil {
			return err
		}
	}
	return nil
}

// Account returns the cached account's preferred username, or "" if none.
func (c *Client) Account(ctx context.Context) (string, error) {
	accounts, err := c.pca.Accounts(ctx)
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", nil
	}
	return accounts[0].PreferredUsername, nil
}

// token acquires an access token silently from the cache.
func (c *Client) token(ctx context.Context) (string, error) {
	accounts, err := c.pca.Accounts(ctx)
	if err != nil {
		return "", err
	}
	if len(accounts) == 0 {
		return "", ErrLoginRequired
	}
	res, err := c.pca.AcquireTokenSilent(ctx, c.scopes, public.WithSilentAccount(accounts[0]))
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrLoginRequired, err)
	}
	return res.AccessToken, nil
}

// GetPresence reads the signed-in user's current presence. It doubles as a
// permission check: a 403 indicates Presence.ReadWrite is not admin-consented.
func (c *Client) GetPresence(ctx context.Context) (availability, activity string, err error) {
	tok, err := c.token(ctx)
	if err != nil {
		return "", "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, graphBase+"/me/presence", nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := c.http.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == http.StatusForbidden {
		return "", "", fmt.Errorf("graph: 403 Forbidden reading presence — Presence.ReadWrite likely lacks admin consent for your tenant")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", "", fmt.Errorf("graph: GET /me/presence returned %s: %s", resp.Status, string(body))
	}
	var out struct {
		Availability string `json:"availability"`
		Activity     string `json:"activity"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", "", err
	}
	return out.Availability, out.Activity, nil
}

// SetPreferredPresence sets a sticky preferred presence for the signed-in user.
func (c *Client) SetPreferredPresence(ctx context.Context, availability, activity, expiration string) error {
	body := map[string]string{
		"availability":       availability,
		"activity":           activity,
		"expirationDuration": expiration,
	}
	return c.post(ctx, "/me/presence/setUserPreferredPresence", body)
}

// ClearPreferredPresence removes any preferred presence, restoring automatic
// presence behavior.
func (c *Client) ClearPreferredPresence(ctx context.Context) error {
	return c.post(ctx, "/me/presence/clearUserPreferredPresence", nil)
}

func (c *Client) post(ctx context.Context, path string, body any) error {
	tok, err := c.token(ctx)
	if err != nil {
		return err
	}
	var buf io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		buf = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, graphBase+path, buf)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+tok)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}
	msg, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
	if resp.StatusCode == http.StatusForbidden {
		return fmt.Errorf("graph: 403 Forbidden calling %s — Presence.ReadWrite likely lacks admin consent for your tenant: %s", path, string(msg))
	}
	return fmt.Errorf("graph: %s returned %s: %s", path, resp.Status, string(msg))
}
