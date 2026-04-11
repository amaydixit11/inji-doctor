package model

import (
	"testing"
)

func TestCheckReport_Compute(t *testing.T) {
	report := &CheckReport{
		Results: []CheckResult{
			{Severity: SeverityOK},
			{Severity: SeverityOK},
			{Severity: SeverityWarning, Fix: "fix it"},
			{Severity: SeverityError, Fix: "fix it"},
			{Severity: SeverityCritical},
			{Severity: SeveritySkipped},
		},
	}

	report.Compute()

	if report.Summary.Total != 6 {
		t.Errorf("Expected Total 6, got %d", report.Summary.Total)
	}
	if report.Summary.Passed != 2 {
		t.Errorf("Expected Passed 2, got %d", report.Summary.Passed)
	}
	if report.Summary.Warnings != 1 {
		t.Errorf("Expected Warnings 1, got %d", report.Summary.Warnings)
	}
	if report.Summary.Failed != 1 {
		t.Errorf("Expected Failed 1, got %d", report.Summary.Failed)
	}
	if report.Summary.Critical != 1 {
		t.Errorf("Expected Critical 1, got %d", report.Summary.Critical)
	}
	if report.Summary.Skipped != 1 {
		t.Errorf("Expected Skipped 1, got %d", report.Summary.Skipped)
	}
	if report.Summary.Fixable != 2 {
		t.Errorf("Expected Fixable 2, got %d", report.Summary.Fixable)
	}
}

func TestCheckReport_WorstSeverity(t *testing.T) {
	tests := []struct {
		results  []CheckResult
		expected Severity
	}{
		{[]CheckResult{{Severity: SeverityOK}}, SeverityOK},
		{[]CheckResult{{Severity: SeverityOK}, {Severity: SeverityWarning}}, SeverityWarning},
		{[]CheckResult{{Severity: SeverityWarning}, {Severity: SeverityError}}, SeverityError},
		{[]CheckResult{{Severity: SeverityError}, {Severity: SeverityCritical}}, SeverityCritical},
	}

	for _, tt := range tests {
		report := &CheckReport{Results: tt.results}
		if got := report.WorstSeverity(); got != tt.expected {
			t.Errorf("WorstSeverity() = %v, want %v", got, tt.expected)
		}
	}
}
