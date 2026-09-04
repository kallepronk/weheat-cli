# weheat-cli

Command-line client for the [WeHeat](https://www.weheat.nl) third-party API, built on
[`weheat-go`](https://github.com/kallepronk/weheat-go). Every command prints JSON.

## Install

```bash
go install github.com/kallepronk/weheat-cli@latest
```

## Auth setup

No registration needed. Log in with your normal WeHeat account (the one from the app or
portal). The CLI uses WeHeat's public `WeheatCommunityAPI` Keycloak client by default.

```bash
export WEHEAT_USERNAME=you@example.com
weheat login            # prompts for the password, stores the token
```

The token lives in `$XDG_CONFIG_HOME/weheat/token.json` (`~/.config/weheat/token.json` on
Linux, `~/Library/Application Support/weheat/token.json` on macOS), mode 0600. Keycloak
rotates refresh tokens; the CLI writes the new one back after every refresh.

Alternatives:

- `WEHEAT_CLIENT_ID=HomeAssistantAPI WEHEAT_CLIENT_SECRET=...` to use the confidential client
  whose credentials WeHeat publishes in its
  [knowledge base](https://support.weheat.nl/s/article/Is-er-een-offici%C3%ABle-Home-Assistant-integratie).
- `WEHEAT_REFRESH_TOKEN=...` for a one-shot run without a stored file.

## Usage

```bash
weheat me
weheat heat-pumps
weheat latest <id>
weheat logs <id> --interval Hour --start 2026-09-01T00:00:00Z --end 2026-09-02T00:00:00Z
weheat energy-logs <id> --interval Day
weheat energy-total <id>
weheat --help
```

`--start`/`--end` default to the last 24 hours. Per-request windows per interval: Minute
2 days, FiveMinute 1 week, FifteenMinute and Hour 1 month, Day 1 year, Week 2 years,
Month 5 years. `raw-logs` covers at most 4 hours.

## Rate limit

WeHeat allows 34 requests per hour per user, shared across every client on that account
(including any collector you run). The CLI blocks at 30/hour per process; `--no-rate-limit`
turns that off.

## Development

```bash
mise install
go test ./...
```

`go.mod` replaces `weheat-go` with the sibling checkout until `weheat-go` v0.1.0 is tagged.

MIT.
