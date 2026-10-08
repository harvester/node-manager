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
			name:     "hostnames preserve order",
			input:    "1.pool.ntp.org 0.pool.ntp.org",
			expected: "1.pool.ntp.org 0.pool.ntp.org",
		},
		{
			name:     "dedupes repeated entries, keeping first occurrence position",
			input:    "0.pool.ntp.org 0.pool.ntp.org 1.pool.ntp.org",
			expected: "0.pool.ntp.org 1.pool.ntp.org",
		},
		{
			name:     "dual-stack IPv4/IPv6/hostname mix preserves configured priority order",
			input:    "fd00::1 0.pool.ntp.org 192.0.2.1 2001:db8::123",
			expected: "fd00::1 0.pool.ntp.org 192.0.2.1 2001:db8::123",
		},
		{
			name:     "same dual-stack set in a different order is not normalized equal, since order is priority",
			input:    "2001:db8::123 192.0.2.1 fd00::1 0.pool.ntp.org",
			expected: "2001:db8::123 192.0.2.1 fd00::1 0.pool.ntp.org",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, NormalizeNTPServers(tt.input))
		})
	}
}

func TestNormalizeNTPServersPreservesPriorityOrder(t *testing.T) {
	a := "fd00::1 0.pool.ntp.org 192.0.2.1"
	b := "192.0.2.1 fd00::1 0.pool.ntp.org"

	// systemd-timesyncd contacts NTP= entries in the given order until one
	// responds, so these must NOT normalize to the same value.
	assert.NotEqual(t, NormalizeNTPServers(a), NormalizeNTPServers(b))
}
