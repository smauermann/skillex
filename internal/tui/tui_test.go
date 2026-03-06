package tui

import (
	"strings"
	"testing"

	"github.com/smauermann/skillex/internal/discovery"
)

func TestProgressBar(t *testing.T) {
	tests := []struct {
		name     string
		fraction float64
		width    int
		wantFull int
	}{
		{"empty", 0.0, 20, 0},
		{"half", 0.5, 20, 10},
		{"full", 1.0, 20, 20},
		{"over", 1.5, 20, 20},
		{"small width", 0.5, 4, 2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bar := progressBar(tt.fraction, tt.width)
			if bar == "" {
				t.Error("expected non-empty progress bar")
			}
		})
	}
}

func TestRenderAnalyticsPanel(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "skill-a", Description: "ALWAYS use this skill.", ActivationStyle: discovery.ActivationDirective, Enabled: true},
		{Name: "skill-b", Description: "Helps with stuff.", ActivationStyle: discovery.ActivationPassive, Enabled: true},
	}

	result := renderAnalyticsPanel(skills[0], skills, 60)
	if result == "" {
		t.Fatal("expected non-empty analytics panel content")
	}
	if !strings.Contains(result, "Activation") {
		t.Error("expected 'Activation' label")
	}
	if !strings.Contains(result, "Description") {
		t.Error("expected 'Description' label")
	}
	if !strings.Contains(result, "Budget") {
		t.Error("expected 'Budget' label")
	}
	if !strings.Contains(result, "Status") {
		t.Error("expected 'Status' label")
	}
	if !strings.Contains(result, "Enabled") {
		t.Error("expected 'Enabled' status for enabled skill")
	}
}

func TestRenderAnalyticsPanelDisabledSkill(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "skill-a", Description: "ALWAYS use this skill.", ActivationStyle: discovery.ActivationDirective, Enabled: true},
		{Name: "skill-b", Description: "Helps with stuff.", ActivationStyle: discovery.ActivationPassive, Enabled: false},
	}

	result := renderAnalyticsPanel(skills[1], skills, 60)
	if !strings.Contains(result, "Disabled") {
		t.Error("expected 'Disabled' status for disabled skill")
	}
	if !strings.Contains(result, "start a new session to apply") {
		t.Error("expected new-session note for disabled skill")
	}
}

func TestBudgetExcludesDisabledSkills(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "a", Description: "AAAAAAAAAA", Enabled: true},  // 10 chars
		{Name: "b", Description: "BBBBBBBBBB", Enabled: false}, // 10 chars, disabled
	}

	total := totalDescChars(skills)
	if total != 10 {
		t.Errorf("expected totalDescChars=10 (excluding disabled), got %d", total)
	}
}

func TestDisabledStats(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "a", Description: "12345", Enabled: true},
		{Name: "b", Description: "1234567890", Enabled: false},
		{Name: "c", Description: "123", Enabled: false},
	}

	count, chars := disabledStats(skills)
	if count != 2 {
		t.Errorf("expected 2 disabled skills, got %d", count)
	}
	if chars != 13 {
		t.Errorf("expected 13 disabled chars, got %d", chars)
	}
}

func TestAnalyticsPanelShowsSavings(t *testing.T) {
	skills := []discovery.Skill{
		{Name: "a", Description: "ALWAYS use this.", Enabled: true, ActivationStyle: discovery.ActivationDirective},
		{Name: "b", Description: "Helps with stuff.", Enabled: false, ActivationStyle: discovery.ActivationPassive},
	}

	result := renderAnalyticsPanel(skills[0], skills, 60)
	if !strings.Contains(result, "disabled skill") {
		t.Error("expected savings line mentioning disabled skills")
	}
	if !strings.Contains(result, "saving") {
		t.Error("expected savings line mentioning 'saving'")
	}
}
