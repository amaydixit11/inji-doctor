// Package checker defines the interface and registry for all diagnostic checks.
// Each check implements the Checker interface and is registered in the global
// registry. The Doctor runs all registered checks and aggregates results.
package checker

import (
	"context"
	"time"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

// Checker is the interface that all diagnostic checks must implement.
type Checker interface {
	// ID returns a unique machine-readable identifier for this check.
	ID() string

	// Name returns a human-readable name for display.
	Name() string

	// Category returns the check category.
	Category() model.CheckCategory

	// Component returns the Inji component this check targets.
	Component() string

	// Run executes the check and returns a result.
	// The context allows for timeout and cancellation.
	Run(ctx context.Context, cfg config.Config) model.CheckResult
}

// SimpleChecker is a helper for checks that can be defined with a function.
type SimpleChecker struct {
	IDFn        func() string
	NameFn      func() string
	CategoryFn  func() model.CheckCategory
	ComponentFn func() string
	RunFn       func(ctx context.Context, cfg config.Config) model.CheckResult
}

func (s SimpleChecker) ID() string                    { return s.IDFn() }
func (s SimpleChecker) Name() string                  { return s.NameFn() }
func (s SimpleChecker) Category() model.CheckCategory { return s.CategoryFn() }
func (s SimpleChecker) Component() string             { return s.ComponentFn() }
func (s SimpleChecker) Run(ctx context.Context, cfg config.Config) model.CheckResult {
	return s.RunFn(ctx, cfg)
}

// Result is a helper for building CheckResults with less boilerplate.
type Result struct {
	Severity   model.Severity
	Message    string
	Detail     string
	Expected   string
	Actual     string
	Fix        string
	FixCommand string
	DocsURL    string
	RawData    map[string]any
}

// WithDetail adds a detail string to the Result.
func (r Result) WithDetail(d string) Result {
	r.Detail = d
	return r
}

// Build creates a CheckResult from a Result builder.
func (r Result) Build(id, name string, category model.CheckCategory, component string) model.CheckResult {
	return model.CheckResult{
		ID:         id,
		Name:       name,
		Category:   category,
		Component:  component,
		Severity:   r.Severity,
		Message:    r.Message,
		Detail:     r.Detail,
		Expected:   r.Expected,
		Actual:     r.Actual,
		Fix:        r.Fix,
		FixCommand: r.FixCommand,
		DocsURL:    r.DocsURL,
		RawData:    r.RawData,
		CheckedAt:  time.Now(),
	}
}

// OK returns a passing Result.
func OK(message string) Result {
	return Result{Severity: model.SeverityOK, Message: message}
}

// Warning returns a warning Result.
func Warning(message, fix string) Result {
	return Result{Severity: model.SeverityWarning, Message: message, Fix: fix}
}

// Error returns an error Result.
func Error(message, fix string) Result {
	return Result{Severity: model.SeverityError, Message: message, Fix: fix}
}

// Critical returns a critical Result.
func Critical(message, fix string) Result {
	return Result{Severity: model.SeverityCritical, Message: message, Fix: fix}
}

// Skipped returns a skipped Result with a reason.
func Skipped(reason string) Result {
	return Result{Severity: model.SeveritySkipped, Message: reason}
}

// All returns the complete list of all registered checkers.
// This is the single source of truth for what checks run.
func All() []Checker {
	return allCheckers()
}

// ForComponent returns only the checkers that target a specific component.
func ForComponent(name string) []Checker {
	var filtered []Checker
	for _, c := range allCheckers() {
		if c.Component() == name || c.Component() == "" {
			filtered = append(filtered, c)
		}
	}
	return filtered
}

// ForCategory returns only the checkers in a specific category.
func ForCategory(cat model.CheckCategory) []Checker {
	var filtered []Checker
	for _, c := range allCheckers() {
		if c.Category() == cat {
			filtered = append(filtered, c)
		}
	}
	return filtered
}
