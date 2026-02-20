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
		{Name: "skill-a", Description: "ALWAYS use this skill.", ActivationStyle: discovery.ActivationDirective},
		{Name: "skill-b", Description: "Helps with stuff.", ActivationStyle: discovery.ActivationPassive},
	}

	result := renderAnalyticsPanel(skills[0], skills, 60)
	if result == "" {
		t.Fatal("expected non-empty analytics panel content")
	}
	if !strings.Contains(result, "Activation") {
		t.Error("expected 'Activation' label")
	}
	if !strings.Contains(result, "Budget") {
		t.Error("expected 'Budget' label")
	}
}
