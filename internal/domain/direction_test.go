package domain

import "testing"

func TestDirectionNormalized(t *testing.T) {
	tests := []struct {
		name string
		in   Direction
		want Direction
	}{
		{name: "trims spaces", in: Direction{From: "  LED ", To: " SVO  "}, want: Direction{From: "LED", To: "SVO"}},
		{name: "already normalized", in: Direction{From: "LED", To: "SVO"}, want: Direction{From: "LED", To: "SVO"}},
		{name: "empty stays empty", in: Direction{From: "   ", To: ""}, want: Direction{From: "", To: ""}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.in.Normalized(); got != tt.want {
				t.Fatalf("expected %+v, got %+v", tt.want, got)
			}
		})
	}
}
