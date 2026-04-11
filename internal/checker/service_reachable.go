package checker

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// serviceReachable checks whether an Inji component's HTTP health endpoint
// is reachable and returning a successful response.
//
// This is the #1 most common failure mode: services not running, wrong port,
// or crash-looping. Based on 19 community posts about Docker Compose failures.
type serviceReachable struct {
	component string
}

func ServiceReachable(component string) Checker {
	return &serviceReachable{component: component}
}

func (c *serviceReachable) ID() string {
	return fmt.Sprintf("service-%s-reachable", strings.ReplaceAll(c.component, "-", ""))
}

func (c *serviceReachable) Name() string {
	comp := model.FindComponent(c.component)
	if comp != nil {
		return fmt.Sprintf("%s is reachable", comp.Label)
	}
	return fmt.Sprintf("%s is reachable", c.component)
}

func (c *serviceReachable) Category() model.CheckCategory {
	return model.CategoryService
}

func (c *serviceReachable) Component() string {
	return c.component
}

func (c *serviceReachable) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	comp := model.FindComponent(c.component)
	if comp == nil {
		return Error(
			fmt.Sprintf("Unknown component: %s", c.component),
			fmt.Sprintf("Check the component name. Known components: %s", componentList()),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	if !cfg.IsEnabled(c.component) {
		return Skipped(fmt.Sprintf("%s is disabled in configuration", comp.Label)).
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	host := cfg.ResolveHost(c.component)
	port := cfg.ResolvePort(c.component, comp.DefaultPort)
	proto := cfg.ResolveProtocol(c.component, comp.Protocol)
	healthPath := cfg.ResolveHealthPath(c.component, comp.HealthPath)

	// For TCP-only services (PostgreSQL, Redis), just check port connectivity.
	if comp.Protocol == "tcp" || proto == "tcp" {
		return c.checkTCP(ctx, host, port, comp, cfg)
	}

	// For HTTP services, hit the health endpoint.
	return c.checkHTTP(ctx, proto, host, port, healthPath, comp, cfg)
}

func (c *serviceReachable) checkTCP(ctx context.Context, host string, port int, comp *model.Component, cfg config.Config) model.CheckResult {
	addr := fmt.Sprintf("%s:%d", host, port)

	dialCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	var d net.Dialer
	conn, err := d.DialContext(dialCtx, "tcp", addr)
	if err != nil {
		return Error(
			fmt.Sprintf("%s is not reachable on %s", comp.Label, addr),
			fmt.Sprintf("Ensure %s is running and listening on port %d. If using Docker Compose, run `docker compose up -d` and wait 30 seconds.",
				comp.Label, port),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}
	conn.Close()

	return OK(fmt.Sprintf("%s is running on %s", comp.Label, addr)).
		Build(c.ID(), c.Name(), c.Category(), c.component)
}

func (c *serviceReachable) checkHTTP(ctx context.Context, proto, host string, port int, healthPath string, comp *model.Component, cfg config.Config) model.CheckResult {
	url := fmt.Sprintf("%s://%s:%d%s", proto, host, port, healthPath)

	reqCtx, cancel := context.WithTimeout(ctx, time.Duration(cfg.TimeoutSec)*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return Error(
			fmt.Sprintf("Failed to create request for %s: %v", comp.Label, err),
			"Check the health endpoint path configuration.",
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	client := &http.Client{
		Timeout: time.Duration(cfg.TimeoutSec) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // Don't follow redirects — we want to see them
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		if strings.Contains(err.Error(), "connection refused") {
			return Error(
				fmt.Sprintf("%s is not running on %s", comp.Label, url),
				fmt.Sprintf("Start %s. If using Docker Compose: `cd <inji-dir> && docker compose up -d %s`",
					comp.Label, c.component),
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}
		if strings.Contains(err.Error(), "context deadline exceeded") || strings.Contains(err.Error(), "timeout") {
			return Error(
				fmt.Sprintf("%s timed out on %s (>%ds)", comp.Label, url, cfg.TimeoutSec),
				fmt.Sprintf("%s may be starting up or stuck. Check logs: `docker compose logs %s`",
					comp.Label, c.component),
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}
		if strings.Contains(err.Error(), "no such host") {
			return Critical(
				fmt.Sprintf("DNS resolution failed for %s", host),
				fmt.Sprintf("Check that '%s' resolves to a valid IP. For local setups, use 'localhost' or '127.0.0.1'.",
					host),
			).Build(c.ID(), c.Name(), c.Category(), c.component)
		}
		return Error(
			fmt.Sprintf("Failed to reach %s: %v", comp.Label, err),
			fmt.Sprintf("Check if %s is running and the URL is correct.", comp.Label),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}
	defer resp.Body.Close()

	// Read a small amount of the body for diagnostics.
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))

	// 200-299 is healthy.
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return OK(fmt.Sprintf("%s is healthy at %s (HTTP %d)", comp.Label, url, resp.StatusCode)).
			Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// 301/302 redirect — useful diagnostic info.
	if resp.StatusCode == 301 || resp.StatusCode == 302 {
		location := resp.Header.Get("Location")
		return Warning(
			fmt.Sprintf("%s redirected (HTTP %d) → %s", comp.Label, resp.StatusCode, location),
			fmt.Sprintf("Follow the redirect or update the base URL. The service may be at: %s", location),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// 404 — endpoint doesn't exist.
	if resp.StatusCode == 404 {
		return Error(
			fmt.Sprintf("%s returned 404 at %s", comp.Label, url),
			fmt.Sprintf("Health endpoint path may be wrong. Expected path: %s. If the service uses a different path, configure it with --health-path.",
				healthPath),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// 500+ — service is running but broken.
	if resp.StatusCode >= 500 {
		bodyPreview := string(body)
		if len(bodyPreview) > 200 {
			bodyPreview = bodyPreview[:200] + "..."
		}
		return Error(
			fmt.Sprintf("%s returned HTTP %d at %s", comp.Label, resp.StatusCode, url),
			fmt.Sprintf("Service is running but returning errors. Check logs: `docker compose logs %s`. Response: %s",
				c.component, bodyPreview),
		).Build(c.ID(), c.Name(), c.Category(), c.component)
	}

	// Other status codes.
	return Warning(
		fmt.Sprintf("%s returned unexpected HTTP %d at %s", comp.Label, resp.StatusCode, url),
		fmt.Sprintf("Check the service logs for details: `docker compose logs %s`", c.component),
	).Build(c.ID(), c.Name(), c.Category(), c.component)
}

// componentList returns a comma-separated list of known component names.
func componentList() string {
	comps := model.KnownComponents()
	names := make([]string, len(comps))
	for i, c := range comps {
		names[i] = c.Name
	}
	return strings.Join(names, ", ")
}
