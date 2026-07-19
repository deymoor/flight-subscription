package kafkacommon

import (
	"reflect"
	"testing"
)

func TestNormalizeBrokers(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{name: "nil", in: nil, want: []string{}},
		{name: "trims and drops empty", in: []string{" a ", "", "  ", "b"}, want: []string{"a", "b"}},
		{name: "already clean", in: []string{"a", "b"}, want: []string{"a", "b"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeBrokers(tt.in)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, got)
			}
		})
	}
}
