package ksmtuned

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestKsmtuned_increase(t *testing.T) {
	tests := []struct {
		name     string
		ksmtuned Ksmtuned
		expected uint
	}{
		{
			name: "normal",
			ksmtuned: Ksmtuned{
				curPage:  100,
				boost:    400,
				maxPages: 99999,
			},
			expected: 500,
		},
		{
			name: "over max pages",
			ksmtuned: Ksmtuned{
				curPage:  200,
				boost:    400,
				maxPages: 500,
			},
			expected: 500,
		},
		{
			name: "equal max pages",
			ksmtuned: Ksmtuned{
				curPage:  100,
				boost:    400,
				maxPages: 500,
			},
			expected: 500,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ksmtuned.increase()
			assert.Equal(t, tt.expected, tt.ksmtuned.curPage, "case %q", tt.name)
		})
	}
}

func TestKsmtuned_decrease(t *testing.T) {
	tests := []struct {
		name     string
		ksmtuned Ksmtuned
		expected uint
	}{
		{
			name: "normal",
			ksmtuned: Ksmtuned{
				curPage:  1000,
				decay:    200,
				minPages: 100,
			},
			expected: 800,
		},
		{
			name: "over min pages",
			ksmtuned: Ksmtuned{
				curPage:  150,
				decay:    200,
				minPages: 100,
			},
			expected: 100,
		},
		{
			name: "equal min pages",
			ksmtuned: Ksmtuned{
				curPage:  150,
				decay:    50,
				minPages: 100,
			},
			expected: 100,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.ksmtuned.decrease()
			assert.Equal(t, tt.expected, tt.ksmtuned.curPage, "case %q", tt.name)
		})
	}
}
