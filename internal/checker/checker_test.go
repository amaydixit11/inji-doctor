package checker

import (
	"context"
	"testing"

	"github.com/inji/inji-doctor/internal/config"
	"github.com/inji/inji-doctor/internal/model"
)

func TestSimpleChecker(t *testing.T) {
	c := SimpleChecker{
		IDFn:        func() string { return "test-id" },
		NameFn:      func() string { return "Test Name" },
		CategoryFn:  func() model.CheckCategory { return model.CategoryService },
		ComponentFn: func() string { return "test-comp" },
		RunFn: func(ctx context.Context, cfg config.Config) model.CheckResult {
			return OK("success").Build("test-id", "Test Name", model.CategoryService, "test-comp")
		},
	}

	if c.ID() != "test-id" {
		t.Errorf("Expected ID test-id, got %s", c.ID())
	}
	if c.Name() != "Test Name" {
		t.Errorf("Expected Name Test Name, got %s", c.Name())
	}

	res := c.Run(context.Background(), config.Config{})
	if res.Severity != model.SeverityOK {
		t.Errorf("Expected Severity OK, got %s", res.Severity)
	}
}

func TestResultHelpers(t *testing.T) {
	res := OK("ok message")
	if res.Severity != model.SeverityOK {
		t.Errorf("OK() severity mismatch")
	}

	res = Warning("warn", "fix")
	if res.Severity != model.SeverityWarning || res.Fix != "fix" {
		t.Errorf("Warning() mismatch")
	}

	res = Error("err", "fix")
	if res.Severity != model.SeverityError {
		t.Errorf("Error() mismatch")
	}

	res = Critical("crit", "fix")
	if res.Severity != model.SeverityCritical {
		t.Errorf("Critical() mismatch")
	}

	res = Skipped("skip")
	if res.Severity != model.SeveritySkipped {
		t.Errorf("Skipped() mismatch")
	}
}
