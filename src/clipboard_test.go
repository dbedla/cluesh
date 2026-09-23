package cluesh

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestShouldCopy(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		redOnly bool
		want    bool
	}{
		{"always red-only command", "always", true, true},
		{"always modifying command", "always", false, true},
		{"never red-only command", "never", true, false},
		{"read-only red-only command", "read-only", true, true},
		{"read-only modifying command", "read-only", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ShouldCopy(tt.mode, tt.redOnly))
		})
	}
}