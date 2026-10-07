package config

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/harvester/node-manager/pkg/apis/node.harvesterhci.io/v1beta1"
)

func TestReGenerateNTPConfig(t *testing.T) {
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
			name:     "dedupes duplicate servers, keeping first occurrence position",
			input:    "0.suse.pool.ntp.org 0.suse.pool.ntp.org 1.suse.pool.ntp.org",
			expected: "0.suse.pool.ntp.org 1.suse.pool.ntp.org",
		},
		{
			name:     "preserves configured priority order for a dual-stack IPv4/IPv6/hostname mix",
			input:    "fd00::1 0.suse.pool.ntp.org 192.0.2.1 2001:db8::123",
			expected: "fd00::1 0.suse.pool.ntp.org 192.0.2.1 2001:db8::123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := reGenerateNTPConfig(&v1beta1.NTPConfig{NTPServers: tt.input})
			assert.Equal(t, tt.expected, got.NTPServers)
		})
	}
}

func TestReGenerateNTPConfigPreservesPriorityOrder(t *testing.T) {
	a := reGenerateNTPConfig(&v1beta1.NTPConfig{NTPServers: "fd00::1 192.0.2.1 0.suse.pool.ntp.org"})
	b := reGenerateNTPConfig(&v1beta1.NTPConfig{NTPServers: "0.suse.pool.ntp.org fd00::1 192.0.2.1"})

	// systemd-timesyncd contacts NTP= entries in the given order until one
	// responds, so reordering must be treated as a real configuration
	// change, not a no-op.
	assert.NotEqual(t, a.NTPServers, b.NTPServers)
}
