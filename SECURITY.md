# Security Policy

## Reporting a vulnerability

Please report security issues privately to **opensource@simtabi.com**. Do not
open a public issue for security problems.

Include where possible: affected version/commit, platform, a description of the
issue, and reproduction steps. We aim to acknowledge reports within a few
business days.

## Scope notes

`ms-teams-activity` stores a Microsoft Graph token cache on disk (mode `0600`)
when the `graph` engine is used, and writes runtime status/override files under
the user cache directory. The synthetic-input engine injects local input events
and, on macOS, holds a power assertion. There is no network listener by default
(the control plane is file-based; an optional loopback API is off unless
`control.port` is set).

## Responsible use

This tool can keep a Teams presence "Available" automatically. It is intended
for personal productivity. Using it to misrepresent availability in violation of
an employer policy or agreement is out of scope and discouraged — see the
"Responsible use" section of the README.
