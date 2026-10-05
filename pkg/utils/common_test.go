package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNormalizeNTPServers(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty",
			input:    "",
			expected: "",
		},
		{
			name:     "single ipv4",
			input:    "192.0.2.1",
			expected: "192.0.2.1",
		},
		{
			name:     "hostnames already sorted",
			input:    "0.pool.ntp.org 1.pool.ntp.org",
			expected: "0.pool.ntp.org 1.pool.ntp.org",
		},
		{
			name:     "dedupes repeated entries",
			input:    "0.pool.ntp.org 0.pool.ntp.org 1.pool.ntp.org",
			expected: "0.pool.ntp.org 1.pool.ntp.org",
		},
		{
			name:     "order independent for dual-stack IPv4/IPv6/hostname mix",
			input:    "fd00::1 0.pool.ntp.org 192.0.2.1 2001:db8::123",
			expected: "0.pool.ntp.org 192.0.2.1 2001:db8::123 fd00::1",
		},
		{
			name:     "same dual-stack set in a different order normalizes equal",
			input:    "2001:db8::123 192.0.2.1 fd00::1 0.pool.ntp.org",
			expected: "0.pool.ntp.org 192.0.2.1 2001:db8::123 fd00::1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeNTPServers(tt.input))
		})
	}
}

func TestNormalizeNTPServersIsOrderIndependent(t *testing.T) {
	a := "fd00::1 0.pool.ntp.org 192.0.2.1"
	b := "192.0.2.1 fd00::1 0.pool.ntp.org"

	assert.Equal(t, NormalizeNTPServers(a), NormalizeNTPServers(b))
}
