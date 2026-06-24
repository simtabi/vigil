# Graph engine

Sets a sticky **preferred presence** via the Microsoft Graph API instead of
faking input. It survives screen lock and needs no GUI session, but requires an
Entra app and **admin-consented** `Presence.ReadWrite`.

Enable with `"engine": "graph"` (or `"both"`).

## How it works

- On each active cycle (throttled to `refresh_minutes`) it calls
  `POST /me/presence/setUserPreferredPresence` with your configured
  `availability`/`activity` and an ISO-8601 `expiration` (e.g. `PT8H`).
- When the schedule ends or the service stops, it calls
  `clearUserPreferredPresence` so your account reverts to automatic presence —
  no stuck "Available".
- Preferred presence only takes effect while a Teams **presence session** exists
  (you're signed in to Teams somewhere). Otherwise presence shows Offline.

## One-time setup

1. **Register an Entra application** (Microsoft Entra admin center → App
   registrations → New registration). Choose accounts in your organization.
2. Under **Authentication**, enable **Allow public client flows** (device-code
   flow uses a public client; no secret is stored).
3. Under **API permissions**, add Microsoft Graph **delegated**
   `Presence.ReadWrite`. This permission **requires admin consent** — have a
   tenant administrator grant it. Most users cannot self-consent.
4. Put the application (client) ID and tenant into `config.json`:

   ```jsonc
   "graph": { "tenant_id": "<tenant-guid-or-common>", "client_id": "<app-id>", ... }
   ```

5. Sign in:

   ```bash
   mta auth login     # follow the device-code prompt
   mta auth status    # confirm the signed-in account
   ```

The token is cached at `os.UserCacheDir()/ms-teams-activity/token.json`
(mode `0600`). Use `mta auth logout` to remove it.

## Troubleshooting

- **403 Forbidden** on set/clear: `Presence.ReadWrite` is not admin-consented
  for your tenant. Ask an admin to grant it.
- **Presence shows Offline**: no Teams presence session — sign in to Teams.

[← Docs index](../../README.md#documentation)
