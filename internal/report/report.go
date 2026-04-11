// Package report handles output formatting for diagnostic reports.
// Supports terminal (colored), JSON, and Markdown output.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/inji/inji-doctor/internal/model"
)

// Terminal renders a report to the terminal with colors and formatting.
type Terminal struct {
	w       io.Writer
	verbose bool
	noColor bool
}

// NewTerminal creates a terminal reporter.
func NewTerminal(w io.Writer, verbose, noColor bool) *Terminal {
	return &Terminal{w: w, verbose: verbose, noColor: noColor}
}

// Write renders the report to the terminal.
func (t *Terminal) Write(report *model.CheckReport) error {
	// Header.
	fmt.Fprintf(t.w, "\n")
	fmt.Fprintf(t.w, "  ┌──────────────────────────────────────────────────────────┐\n")
	fmt.Fprintf(t.w, "  │              Inji Doctor — Stack Health Check            │\n")
	fmt.Fprintf(t.w, "  └──────────────────────────────────────────────────────────┘\n")
	fmt.Fprintf(t.w, "\n")

	// Stack profile and timing.
	fmt.Fprintf(t.w, "  Stack: %s\n", report.StackProfile)
	fmt.Fprintf(t.w, "  Time:  %s (%.1fs)\n", report.StartedAt.Format(time.RFC3339), report.Duration.Seconds())
	fmt.Fprintf(t.w, "\n")

	// Results grouped by category.
	categories := t.groupByCategory(report.Results)

	for _, cat := range categories {
		t.renderCategory(cat)
	}

	// Summary.
	t.renderSummary(report)

	return nil
}

func (t *Terminal) renderCategory(results []model.CheckResult) {
	if len(results) == 0 {
		return
	}

	// Category header.
	catName := results[0].Category
	fmt.Fprintf(t.w, "  ── %s ──\n", strings.ToUpper(string(catName)))

	for _, r := range results {
		status := r.Status()
		name := r.Name

		// Severity color via ANSI codes (simplified — no external dep).
		var colorCode, reset string
		if !t.noColor {
			switch r.Severity {
			case model.SeverityOK:
				colorCode, reset = "\033[32m", "\033[0m" // green
			case model.SeverityWarning:
				colorCode, reset = "\033[33m", "\033[0m" // yellow
			case model.SeverityError, model.SeverityCritical:
				colorCode, reset = "\033[31m", "\033[0m" // red
			case model.SeveritySkipped:
				colorCode, reset = "\033[90m", "\033[0m" // gray
			}
		}

		fmt.Fprintf(t.w, "  %s%s %s%s\n", colorCode, status, name, reset)

		if !r.IsPassing() {
			indent := "     "
			fmt.Fprintf(t.w, "%s%s%s%s\n", indent, colorCode, r.Message, reset)

			if r.Fix != "" {
				fmt.Fprintf(t.w, "%s→ Fix: %s\n", indent, r.Fix)
			}

			if r.FixCommand != "" {
				fmt.Fprintf(t.w, "%s  $ %s\n", indent, r.FixCommand)
			}
		}

		if t.verbose && r.Detail != "" {
			fmt.Fprintf(t.w, "     Detail: %s\n", r.Detail)
		}
		if t.verbose && r.Duration > 0 {
			fmt.Fprintf(t.w, "     Took: %s\n", r.Duration.Round(time.Millisecond))
		}
	}

	fmt.Fprintf(t.w, "\n")
}

func (t *Terminal) renderSummary(report *model.CheckReport) {
	s := report.Summary

	fmt.Fprintf(t.w, "  ── SUMMARY ──\n")
	fmt.Fprintf(t.w, "  Total: %d  |  Passed: %d  |  Warnings: %d  |  Failed: %d  |  Critical: %d  |  Skipped: %d\n",
		s.Total, s.Passed, s.Warnings, s.Failed, s.Critical, s.Skipped)

	if s.Fixable > 0 {
		fmt.Fprintf(t.w, "  %d issue(s) have suggested fixes.\n", s.Fixable)
	}

	fmt.Fprintf(t.w, "\n")

	// Overall status.
	worst := report.WorstSeverity()
	var overallMsg, colorCode, reset string
	if !t.noColor {
		reset = "\033[0m"
	}
	switch worst {
	case model.SeverityOK:
		overallMsg = "✅ All checks passed — Inji stack is healthy"
		if !t.noColor {
			colorCode = "\033[32m"
		}
	case model.SeverityWarning:
		overallMsg = "⚠️  Warnings found — stack is functional but has issues"
		if !t.noColor {
			colorCode = "\033[33m"
		}
	case model.SeverityError:
		overallMsg = "❌ Errors found — some components are not working correctly"
		if !t.noColor {
			colorCode = "\033[31m"
		}
	case model.SeverityCritical:
		overallMsg = "🚨 Critical issues found — stack may be non-functional"
		if !t.noColor {
			colorCode = "\033[31m\033[1m"
		}
	default:
		overallMsg = "Checks completed"
	}

	fmt.Fprintf(t.w, "  %s%s%s\n", colorCode, overallMsg, reset)
	fmt.Fprintf(t.w, "\n")
}

func (t *Terminal) groupByCategory(results []model.CheckResult) [][]model.CheckResult {
	// Define the display order for categories.
	order := []model.CheckCategory{
		model.CategoryService,
		model.CategoryConfig,
		model.CategoryKeys,
		model.CategoryTrust,
		model.CategoryDatabase,
		model.CategoryCache,
		model.CategoryIntegration,
		model.CategoryDocker,
		model.CategoryNetwork,
		model.CategorySystem,
	}

	// Group results.
	groups := make(map[model.CheckCategory][]model.CheckResult)
	for _, r := range results {
		groups[r.Category] = append(groups[r.Category], r)
	}

	// Return in order.
	var out [][]model.CheckResult
	for _, cat := range order {
		if group, ok := groups[cat]; ok {
			out = append(out, group)
		}
	}

	return out
}

// JSON renders a report as JSON.
type JSON struct {
	w io.Writer
}

func NewJSON(w io.Writer) *JSON {
	return &JSON{w: w}
}

func (j *JSON) Write(report *model.CheckReport) error {
	enc := json.NewEncoder(j.w)
	enc.SetIndent("", "  ")
	return enc.Encode(report)
}

// Markdown renders a report as Markdown.
type Markdown struct {
	w io.Writer
}

func NewMarkdown(w io.Writer) *Markdown {
	return &Markdown{w: w}
}

func (m *Markdown) Write(report *model.CheckReport) error {
	fmt.Fprintf(m.w, "# Inji Doctor — Stack Health Check\n\n")
	fmt.Fprintf(m.w, "- **Stack:** %s\n", report.StackProfile)
	fmt.Fprintf(m.w, "- **Time:** %s\n", report.StartedAt.Format(time.RFC3339))
	fmt.Fprintf(m.w, "- **Duration:** %s\n\n", report.Duration.Round(time.Millisecond))

	// Summary table.
	s := report.Summary
	fmt.Fprintf(m.w, "## Summary\n\n")
	fmt.Fprintf(m.w, "| Total | Passed | Warnings | Failed | Critical | Skipped |\n")
	fmt.Fprintf(m.w, "|-------|--------|----------|--------|----------|---------|\n")
	fmt.Fprintf(m.w, "| %d | %d | %d | %d | %d | %d |\n\n",
		s.Total, s.Passed, s.Warnings, s.Failed, s.Critical, s.Skipped)

	// Results by category.
	fmt.Fprintf(m.w, "## Results\n\n")

	categories := make(map[model.CheckCategory][]model.CheckResult)
	for _, r := range report.Results {
		categories[r.Category] = append(categories[r.Category], r)
	}

	for cat, results := range categories {
		fmt.Fprintf(m.w, "### %s\n\n", strings.ToUpper(string(cat)))
		fmt.Fprintf(m.w, "| Status | Check | Message | Fix |\n")
		fmt.Fprintf(m.w, "|--------|-------|---------|-----|\n")

		for _, r := range results {
			status := r.Status()
			fix := r.Fix
			if fix == "" {
				fix = "—"
			}
			msg := r.Message
			if msg == "" {
				msg = "—"
			}
			fmt.Fprintf(m.w, "| %s | %s | %s | %s |\n", status, r.Name, msg, fix)
		}

		fmt.Fprintf(m.w, "\n")
	}

	// Issues needing attention.
	var issues []model.CheckResult
	for _, r := range report.Results {
		if r.IsFailing() {
			issues = append(issues, r)
		}
	}

	if len(issues) > 0 {
		fmt.Fprintf(m.w, "## Issues Requiring Attention\n\n")
		for _, r := range issues {
			fmt.Fprintf(m.w, "### %s %s\n\n", r.Status(), r.Name)
			if r.Message != "" {
				fmt.Fprintf(m.w, "**Issue:** %s\n\n", r.Message)
			}
			if r.Detail != "" {
				fmt.Fprintf(m.w, "**Detail:**\n```\n%s\n```\n\n", r.Detail)
			}
			if r.Fix != "" {
				fmt.Fprintf(m.w, "**Fix:** %s\n\n", r.Fix)
			}
			if r.FixCommand != "" {
				fmt.Fprintf(m.w, "```bash\n%s\n```\n\n", r.FixCommand)
			}
			if r.DocsURL != "" {
				fmt.Fprintf(m.w, "[📖 Documentation](%s)\n\n", r.DocsURL)
			}
		}
	}

	return nil
}
