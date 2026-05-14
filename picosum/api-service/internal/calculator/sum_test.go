package calculator_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/picosum/api-service/internal/calculator"
)

func TestAdd(t *testing.T) {
	tests := []struct {
		name string
		x, y int
		want int
	}{
		{"5+3", 5, 3, 8},
		{"0+0", 0, 0, 0},
		{"10+10", 10, 10, 20},
		{"négatifs", -1, -2, -3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, calculator.Add(tt.x, tt.y))
		})
	}
}
