// Package report handles output formatting for diagnostic reports.
// Supports terminal (colored), JSON, and Markdown output.
package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
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
	if noColor {
		color.NoColor = true
	}
	return &Terminal{w: w, verbose: verbose, noColor: noColor}
}

// Write renders the report to the terminal.
func (t *Terminal) Write(report *model.CheckReport) error {
	t.renderBanner()

	// Stack profile and timing.
	cyan := color.New(color.FgCyan).SprintFunc()
	white := color.New(color.FgWhite, color.Bold).SprintFunc()
	fmt.Fprintf(t.w, "  %s  %s\n", cyan("Stack:"), white(report.StackProfile))
	fmt.Fprintf(t.w, "  %s  %s (%s)\n", cyan("Time: "), report.StartedAt.Format("2006-01-02 15:04:05"), report.Duration.Round(time.Millisecond))

	// Environment info.
	hostname, _ := os.Hostname()
	envStyle := color.New(color.FgHiBlack).SprintFunc()
	fmt.Fprintf(t.w, "  %s  %s\n",
		cyan("Env:  "), envStyle(fmt.Sprintf("%s | %s | %d core(s)", hostname, runtime.GOOS, runtime.NumCPU())))
	fmt.Fprintf(t.w, "\n")

	// Results grouped by category.
	categories := t.groupByCategory(report.Results)

	for _, cat := range categories {
		t.renderCategory(cat)
	}

	// Summary.
	t.renderSummary(report)

	// Doctor's Advice.
	t.renderAdvice(report)

	return nil
}

func (t *Terminal) renderBanner() {
	if t.noColor {
		banner := `
  Inji Doctor — Stack Health Check
  ──────────────────────────────
`
		fmt.Fprint(t.w, banner)
		return
	}

	logo := []string{
		`  ██╗███╗   ██╗██╗██╗      ███╗   ███╗ ██████╗ ███████╗██╗██████╗ `,
		`  ██║████╗  ██║██║██║      ████╗ ████║██╔═══██╗██╔════╝██║██╔══██╗`,
		`  ██║██╔██╗ ██║██║██║      ██╔████╔██║██║   ██║███████╗██║██████╔╝`,
		`  ██║██║╚██╗██║██║██║      ██║╚██╔╝██║██║   ██║╚════██║██║██╔═══╝ `,
		`  ██║██║ ╚████║██║██║ ██╗  ██║ ╚═╝ ██║╚██████╔╝███████║██║██║     `,
		`  ╚═╝╚═╝  ╚═══╝╚═╝╚═╝ ╚═╝  ╚═╝     ╚═╝ ╚═════╝ ╚══════╝╚═╝╚═╝     `,
	}

	// MOSIP Gradient colors: Deep Blue -> Bright Blue -> Orange
	deepBlue := RGB{R: 0, G: 70, B: 160}
	brightBlue := RGB{R: 0, G: 160, B: 255}
	orange := RGB{R: 255, G: 140, B: 0}

	for _, line := range logo {
		t.printGradient(line, deepBlue, brightBlue, orange)
	}

	versionStyle := color.New(color.FgHiBlack).SprintFunc()
	titleStyle := color.New(color.FgHiBlue, color.Bold).SprintFunc()
	boxStyle := color.New(color.FgHiBlack).SprintFunc()

	fmt.Fprintf(t.w, "  %s  %s\n", titleStyle("INJI DEVELOPER TOOLKIT"), versionStyle("v0.1.0"))
	fmt.Fprintf(t.w, "  %s\n\n", boxStyle("──────────────────────────────────────────────────────────────────"))
}

type RGB struct {
	R, G, B int
}

func (t *Terminal) printGradient(text string, start, mid, end RGB) {
	runes := []rune(text)
	total := len(runes)
	if total == 0 {
		fmt.Fprintln(t.w)
		return
	}

	for i, r := range runes {
		var color RGB
		ratio := float64(i) / float64(total)

		if ratio < 0.5 {
			// Interpolate between start and mid
			localRatio := ratio * 2
			color.R = int(float64(start.R) + localRatio*float64(mid.R-start.R))
			color.G = int(float64(start.G) + localRatio*float64(mid.G-start.G))
			color.B = int(float64(start.B) + localRatio*float64(mid.B-start.B))
		} else {
			// Interpolate between mid and end
			localRatio := (ratio - 0.5) * 2
			color.R = int(float64(mid.R) + localRatio*float64(end.R-mid.R))
			color.G = int(float64(mid.G) + localRatio*float64(end.G-mid.G))
			color.B = int(float64(mid.B) + localRatio*float64(end.B-mid.B))
		}

		fmt.Fprintf(t.w, "\x1b[38;2;%d;%d;%dm%c\x1b[0m", color.R, color.G, color.B, r)
	}
	fmt.Fprintln(t.w)
}

func (t *Terminal) renderCategory(results []model.CheckResult) {
	if len(results) == 0 {
		return
	}

	// Category header.
	catName := strings.ToUpper(string(results[0].Category))
	catStyle := color.New(color.FgHiBlack, color.Bold).SprintFunc()
	fmt.Fprintf(t.w, "  %s\n", catStyle(catName))

	for _, r := range results {
		status := r.Status()
		name := r.Name

		var icon string
		var style *color.Color

		switch r.Severity {
		case model.SeverityOK:
			icon = "●"
			style = color.New(color.FgGreen)
		case model.SeverityWarning:
			icon = "▲"
			style = color.New(color.FgYellow)
		case model.SeverityError, model.SeverityCritical:
			icon = "✖"
			style = color.New(color.FgRed)
		case model.SeveritySkipped:
			icon = "○"
			style = color.New(color.FgHiBlack)
		}

		if t.noColor {
			fmt.Fprintf(t.w, "  %s %s\n", status, name)
		} else {
			fmt.Fprintf(t.w, "  %s %-30s %s\n", style.Sprint(icon), name, style.Sprint(status))
		}

		if !r.IsPassing() {
			indent := "      "
			msgStyle := color.New(color.FgHiWhite).SprintFunc()
			if r.Severity == model.SeverityCritical || r.Severity == model.SeverityError {
				msgStyle = color.New(color.FgRed).SprintFunc()
			} else if r.Severity == model.SeverityWarning {
				msgStyle = color.New(color.FgYellow).SprintFunc()
			}

			fmt.Fprintf(t.w, "%s%s\n", indent, msgStyle(r.Message))

			if r.Fix != "" {
				fixStyle := color.New(color.FgCyan).SprintFunc()
				fmt.Fprintf(t.w, "%s%s %s\n", indent, fixStyle("→ Fix:"), r.Fix)
			}

			if r.FixCommand != "" {
				cmdStyle := color.New(color.FgHiBlack).SprintFunc()
				fmt.Fprintf(t.w, "%s  %s\n", indent, cmdStyle("$ "+r.FixCommand))
			}
		}

		if t.verbose && r.Detail != "" {
			fmt.Fprintf(t.w, "      Detail: %s\n", r.Detail)
		}
		if t.verbose && r.Duration > 0 {
			fmt.Fprintf(t.w, "      Took: %s\n", r.Duration.Round(time.Millisecond))
		}
	}

	fmt.Fprintf(t.w, "\n")
}

func (t *Terminal) renderAdvice(report *model.CheckReport) {
	if t.noColor {
		return
	}

	score := report.HealthScore()
	adviceStyle := color.New(color.FgHiCyan, color.Italic).SprintFunc()
	headerStyle := color.New(color.FgHiYellow, color.Bold).SprintFunc()

	var msg string
	switch {
	case score >= 90:
		msg = "Your stack is in top shape! Ready for high-volume credential orchestration."
	case score >= 70:
		msg = "Looking good, but keep an eye on those warnings to ensure stability."
	case score >= 40:
		msg = "The stack is functional but brittle. Prioritize fixing the failed services."
	default:
		msg = "The stack needs immediate attention. Start by ensuring Docker is running."
	}

	fmt.Fprintf(t.w, "  %s %s\n", headerStyle("DOCTOR'S ADVICE:"), adviceStyle(msg))
	fmt.Fprintf(t.w, "\n")
}

func (t *Terminal) renderSummary(report *model.CheckReport) {
	s := report.Summary

	lineStyle := color.New(color.FgHiBlack).SprintFunc()
	fmt.Fprintf(t.w, "  %s\n", lineStyle("────────────────────────────────────────"))

	// Grouped summary stats
	success := color.New(color.FgGreen).SprintFunc()
	warn := color.New(color.FgYellow).SprintFunc()
	fail := color.New(color.FgRed).SprintFunc()
	gray := color.New(color.FgHiBlack).SprintFunc()

	fmt.Fprintf(t.w, "  %s %d  |  %s %d  |  %s %d  |  %s %d  |  %s %d\n",
		success("Passed:"), s.Passed,
		warn("Warnings:"), s.Warnings,
		fail("Failed:"), s.Failed,
		fail("Critical:"), s.Critical,
		gray("Skipped:"), s.Skipped)

	// Health Score
	score := report.HealthScore()
	var scoreColor *color.Color
	switch {
	case score >= 90:
		scoreColor = color.New(color.FgGreen, color.Bold)
	case score >= 70:
		scoreColor = color.New(color.FgYellow, color.Bold)
	default:
		scoreColor = color.New(color.FgRed, color.Bold)
	}
	fmt.Fprintf(t.w, "  %s %s\n", lineStyle("Health Index:"), scoreColor.Sprintf(" %d%%", score))

	if s.Fixable > 0 {
		cyan := color.New(color.FgCyan).SprintFunc()
		fmt.Fprintf(t.w, "  %s %d issue(s) have suggested fixes.\n", cyan("💡"), s.Fixable)
	}

	fmt.Fprintf(t.w, "\n")

	// Overall status.
	worst := report.WorstSeverity()
	var icon, overallMsg string
	var style *color.Color

	switch worst {
	case model.SeverityOK:
		icon = "✅"
		overallMsg = "All checks passed — Inji stack is healthy"
		style = color.New(color.FgGreen, color.Bold)
	case model.SeverityWarning:
		icon = "⚠️ "
		overallMsg = "Warnings found — stack is functional but has issues"
		style = color.New(color.FgYellow, color.Bold)
	case model.SeverityError:
		icon = "❌"
		overallMsg = "Errors found — some components represent risks"
		style = color.New(color.FgRed, color.Bold)
	case model.SeverityCritical:
		icon = "🚨"
		overallMsg = "Critical issues found — stack may be non-functional"
		style = color.New(color.FgRed, color.Bold, color.Underline)
	default:
		overallMsg = "Checks completed"
		style = color.New(color.FgWhite)
	}

	fmt.Fprintf(t.w, "  %s  %s\n", icon, style.Sprint(overallMsg))
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
