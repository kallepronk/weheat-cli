// Command weheat is a thin CLI over github.com/kallepronk/weheat-go. Output is JSON.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/kallepronk/weheat-cli/internal/tokenfile"
	"github.com/kallepronk/weheat-go"
	"golang.org/x/oauth2"
	"golang.org/x/term"
)

const usage = `usage: weheat [flags] <command> [args]

commands:
  login                        password grant; stores the token for later runs
  me                           authenticated user
  heat-pumps                   list heat pumps
  heat-pump <id>               one heat pump
  latest <id>                  latest raw log plus online flag
  logs <id>                    aggregated logs (see --start, --end, --interval)
  raw-logs <id>                unaggregated logs, window at most 4h
  energy-logs <id>             energy buckets (see --start, --end, --interval)
  energy-total <id>            lifetime energy totals

flags:
`

const envHelp = `
environment:
  WEHEAT_CLIENT_ID                         Keycloak client (default WeheatCommunityAPI, public)
  WEHEAT_CLIENT_SECRET                     only for confidential clients such as HomeAssistantAPI
  WEHEAT_USERNAME, WEHEAT_PASSWORD         for login (your WeHeat account); password is prompted when unset
  WEHEAT_REFRESH_TOKEN                     one-shot alternative to the stored token
  WEHEAT_BASE_URL                          override the API root
`

const defaultWindow = 24 * time.Hour

// options is everything the command line and environment contribute.
type options struct {
	start, end time.Time
	interval   string
	tokenPath  string
	noLimit    bool
	baseURL    string
}

func main() {
	if err := run(os.Args[1:], os.Stdout, os.Stderr); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			os.Exit(2)
		}
		fmt.Fprintln(os.Stderr, "weheat:", err)
		os.Exit(1)
	}
}

func run(args []string, stdout, stderr io.Writer) error {
	fs := flag.NewFlagSet("weheat", flag.ContinueOnError)
	fs.SetOutput(stderr)
	fs.Usage = func() {
		fmt.Fprint(stderr, usage)
		fs.PrintDefaults()
		fmt.Fprint(stderr, envHelp)
	}
	var opts options
	var start, end string
	fs.StringVar(&start, "start", "", "range start, RFC 3339 (default: --end minus 24h)")
	fs.StringVar(&end, "end", "", "range end, RFC 3339 (default: now)")
	fs.StringVar(&opts.interval, "interval", "Hour", "Minute|FiveMinute|FifteenMinute|Hour|Day|Week|Month|Year")
	fs.StringVar(&opts.tokenPath, "token-file", "", "where the token is stored (default: $XDG_CONFIG_HOME/weheat/token.json)")
	fs.BoolVar(&opts.noLimit, "no-rate-limit", false, "disable the built-in 30 requests/hour limiter")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() == 0 {
		fs.Usage()
		return flag.ErrHelp
	}
	var err error
	if opts.start, opts.end, err = parseRange(start, end, time.Now()); err != nil {
		return err
	}
	if opts.tokenPath == "" {
		if opts.tokenPath, err = tokenfile.DefaultPath(); err != nil {
			return err
		}
	}
	opts.baseURL = os.Getenv("WEHEAT_BASE_URL")

	ctx := context.Background()
	clientID := os.Getenv("WEHEAT_CLIENT_ID")
	if clientID == "" {
		clientID = weheat.CommunityClientID
	}
	cfg := weheat.OAuthConfig(clientID, os.Getenv("WEHEAT_CLIENT_SECRET"))

	cmd, rest := fs.Arg(0), fs.Args()[1:]
	if cmd == "login" {
		return login(ctx, cfg, opts.tokenPath, stderr)
	}
	client, err := newClient(ctx, cfg, opts, stderr)
	if err != nil {
		return err
	}
	out, err := dispatch(ctx, client, cmd, rest, opts)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

// parseRange fills in defaults: end = now, start = end - 24h.
func parseRange(start, end string, now time.Time) (time.Time, time.Time, error) {
	e := now
	if end != "" {
		var err error
		if e, err = time.Parse(time.RFC3339, end); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("--end: %w", err)
		}
	}
	s := e.Add(-defaultWindow)
	if start != "" {
		var err error
		if s, err = time.Parse(time.RFC3339, start); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("--start: %w", err)
		}
	}
	if !s.Before(e) {
		return time.Time{}, time.Time{}, errors.New("--start must be before --end")
	}
	return s, e, nil
}

func dispatch(ctx context.Context, c *weheat.Client, cmd string, args []string, opts options) (any, error) {
	needID := func() (string, error) {
		if len(args) != 1 {
			return "", fmt.Errorf("%s takes exactly one <id>", cmd)
		}
		return args[0], nil
	}
	interval := weheat.Interval(opts.interval)
	switch cmd {
	case "me":
		return c.Me(ctx)
	case "heat-pumps":
		return c.HeatPumps(ctx, nil)
	case "heat-pump":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.HeatPump(ctx, id)
	case "latest":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.LatestLog(ctx, id)
	case "logs":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.Logs(ctx, id, opts.start, opts.end, interval)
	case "raw-logs":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.RawLogs(ctx, id, opts.start, opts.end)
	case "energy-logs":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.EnergyLogs(ctx, id, opts.start, opts.end, interval)
	case "energy-total":
		id, err := needID()
		if err != nil {
			return nil, err
		}
		return c.EnergyTotal(ctx, id)
	}
	return nil, fmt.Errorf("unknown command %q", cmd)
}

// login runs the password grant and stores the resulting token.
func login(ctx context.Context, cfg *oauth2.Config, tokenPath string, stderr io.Writer) error {
	user := os.Getenv("WEHEAT_USERNAME")
	if user == "" {
		return errors.New("set WEHEAT_USERNAME")
	}
	pass := os.Getenv("WEHEAT_PASSWORD")
	if pass == "" {
		fmt.Fprintf(stderr, "password for %s: ", user)
		b, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Fprintln(stderr)
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		pass = string(b)
	}
	tok, err := cfg.PasswordCredentialsToken(ctx, user, pass)
	if err != nil {
		return err
	}
	if err := tokenfile.Save(tokenPath, tok); err != nil {
		return err
	}
	fmt.Fprintln(stderr, "token stored at", tokenPath)
	return nil
}

// newClient builds a client whose token source writes rotated refresh tokens back to disk.
func newClient(ctx context.Context, cfg *oauth2.Config, opts options, stderr io.Writer) (*weheat.Client, error) {
	var seed *oauth2.Token
	if rt := os.Getenv("WEHEAT_REFRESH_TOKEN"); rt != "" {
		seed = &oauth2.Token{RefreshToken: rt}
	} else {
		var err error
		if seed, err = tokenfile.Load(opts.tokenPath); err != nil {
			return nil, err
		}
	}
	ts := weheat.NotifyTokenSource(cfg.TokenSource(ctx, seed), func(tok *oauth2.Token) {
		if err := tokenfile.Save(opts.tokenPath, tok); err != nil {
			fmt.Fprintln(stderr, "weheat: could not store rotated token:", err)
		}
	})
	var clientOpts []weheat.Option
	if opts.baseURL != "" {
		clientOpts = append(clientOpts, weheat.WithBaseURL(strings.TrimRight(opts.baseURL, "/")))
	}
	if opts.noLimit {
		clientOpts = append(clientOpts, weheat.WithRateLimiter(nil))
	}
	return weheat.NewClient(ts, clientOpts...), nil
}
