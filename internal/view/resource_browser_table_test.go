package view

import (
	"testing"

	"github.com/clawscli/claws/internal/render"
)

func TestCalculateColumnWidths_FirstColumnGrowsToFitContent(t *testing.T) {
	cols := []render.Column{
		{Name: "NAME", Width: 20},
		{Name: "TYPE", Width: 10},
	}

	tests := []struct {
		name         string
		width        int
		contentWidth int
		want         []int
	}{
		{"content fits declared width", 100, 15, []int{3, 20, 77}},
		{"grows into spare width", 100, 40, []int{3, 41, 56}},
		{"capped by available width", 50, 80, []int{3, 37, 10}},
		{"no spare width", 20, 80, []int{3, 20, 10}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ResourceBrowser{width: tt.width}

			got := r.calculateColumnWidths(cols, false, false, false, 3, tt.contentWidth)

			for i := range tt.want {
				if got[i] != tt.want[i] {
					t.Fatalf("widths = %v, want %v", got, tt.want)
				}
			}
		})
	}
}
