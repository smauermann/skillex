package tui

import "testing"

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
