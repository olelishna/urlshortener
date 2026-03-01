package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateShortURL(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "#1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got := GenerateShortURL()

			assert.IsType(t, "string", got)
			assert.Equal(t, len(got), 8)
		})
	}
}
