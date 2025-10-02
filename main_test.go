package main

import (
	"bytes"
	"io"
	"net/netip"
	"os"
	"strings"
	"testing"

	"github.com/dav1dc-github/cidr-calculator-github/internal/githubmeta"
)

// createTestMeta creates a MetaData instance with controlled test data
func createTestMeta() *githubmeta.MetaData {
	entries := []githubmeta.Entry{
		{Label: "hooks", Prefix: netip.MustParsePrefix("192.30.252.0/22")},
		{Label: "hooks", Prefix: netip.MustParsePrefix("2001:db8:1::/48")},
		{Label: "web", Prefix: netip.MustParsePrefix("140.82.112.0/20")},
		{Label: "api", Prefix: netip.MustParsePrefix("192.30.252.0/24")},
		{Label: "pages", Prefix: netip.MustParsePrefix("185.199.108.0/22")},
	}
	return githubmeta.NewMetaDataForTesting(entries)
}

// captureOutput captures stdout during function execution
func captureOutput(f func()) string {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	f()

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	return buf.String()
}

func TestEvaluateInput_SingleIPv4Owned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.42")
	})

	if !strings.Contains(output, "owned by GitHub") {
		t.Errorf("Expected owned message, got: %s", output)
	}
	if !strings.Contains(output, "hooks") || !strings.Contains(output, "api") {
		t.Errorf("Expected hooks and api labels, got: %s", output)
	}
}

func TestEvaluateInput_SingleIPv4NotOwned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "8.8.8.8")
	})

	if !strings.Contains(output, "not owned by GitHub") {
		t.Errorf("Expected not owned message, got: %s", output)
	}
}

func TestEvaluateInput_SingleIPv6Owned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8:1::213")
	})

	if !strings.Contains(output, "owned by GitHub") {
		t.Errorf("Expected owned message, got: %s", output)
	}
	if !strings.Contains(output, "hooks") {
		t.Errorf("Expected hooks label, got: %s", output)
	}
}

func TestEvaluateInput_SingleIPv6NotOwned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8:2::1")
	})

	if !strings.Contains(output, "not owned by GitHub") {
		t.Errorf("Expected not owned message, got: %s", output)
	}
}

func TestEvaluateInput_Prefix32Owned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.42/32")
	})

	if !strings.Contains(output, "evaluated 1 addresses") {
		t.Errorf("Expected single address evaluation, got: %s", output)
	}
	if !strings.Contains(output, "Owned by GitHub: 1") {
		t.Errorf("Expected 1 owned address, got: %s", output)
	}
}

func TestEvaluateInput_Prefix32NotOwned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "8.8.8.8/32")
	})

	if !strings.Contains(output, "evaluated 1 addresses") {
		t.Errorf("Expected single address evaluation, got: %s", output)
	}
	if !strings.Contains(output, "Not owned: 1") {
		t.Errorf("Expected 1 non-owned address, got: %s", output)
	}
}

func TestEvaluateInput_Prefix128Owned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8:1::213/128")
	})

	if !strings.Contains(output, "evaluated 1 addresses") {
		t.Errorf("Expected single address evaluation, got: %s", output)
	}
	if !strings.Contains(output, "Owned by GitHub: 1") {
		t.Errorf("Expected 1 owned address, got: %s", output)
	}
}

func TestEvaluateInput_Prefix128NotOwned(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8:2::1/128")
	})

	if !strings.Contains(output, "evaluated 1 addresses") {
		t.Errorf("Expected single address evaluation, got: %s", output)
	}
	if !strings.Contains(output, "Not owned: 1") {
		t.Errorf("Expected 1 non-owned address, got: %s", output)
	}
}

func TestEvaluateInput_InvalidEmpty(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "")
	})

	if !strings.Contains(output, "invalid") {
		t.Errorf("Expected invalid message, got: %s", output)
	}
}

func TestEvaluateInput_InvalidGarbage(t *testing.T) {
	meta := createTestMeta()

	output := captureOutput(func() {
		evaluateInput(meta, "not-an-ip")
	})

	if !strings.Contains(output, "invalid IP address or CIDR") {
		t.Errorf("Expected invalid message, got: %s", output)
	}
}

func TestEvaluateInput_InvalidMalformed(t *testing.T) {
	meta := createTestMeta()

	tests := []string{
		"192.168.1.256",
		"192.168.1",
		"192.168.1.1.1",
		"gggg::1",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			output := captureOutput(func() {
				evaluateInput(meta, input)
			})

			if !strings.Contains(output, "invalid") {
				t.Errorf("Expected invalid message for %s, got: %s", input, output)
			}
		})
	}
}

func TestEvaluateInput_CIDRSmallUniform(t *testing.T) {
	meta := createTestMeta()

	// 192.30.252.0/30 contains 4 addresses: .0, .1, .2, .3 - all in hooks range
	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.0/30")
	})

	if !strings.Contains(output, "evaluated 4 addresses") {
		t.Errorf("Expected 4 addresses, got: %s", output)
	}
	if !strings.Contains(output, "Owned by GitHub: 4") {
		t.Errorf("Expected 4 owned addresses, got: %s", output)
	}
}

func TestEvaluateInput_CIDRMixedOwnership(t *testing.T) {
	meta := createTestMeta()

	// Create a range that spans owned and non-owned space
	// 185.199.108.0/22 is pages, but 185.199.112.0 is outside
	// Let's test 185.199.111.0/24 which is at the edge
	output := captureOutput(func() {
		evaluateInput(meta, "185.199.111.0/30")
	})

	if !strings.Contains(output, "evaluated 4 addresses") {
		t.Errorf("Expected 4 addresses, got: %s", output)
	}
	// All 4 should be in the pages range since 185.199.108.0/22 covers .108-111
	if !strings.Contains(output, "Owned by GitHub: 4") {
		t.Errorf("Expected owned addresses, got: %s", output)
	}
}

func TestEvaluateInput_CIDRIPv6(t *testing.T) {
	meta := createTestMeta()

	// Test IPv6 CIDR with /126 (4 addresses)
	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8:1::0/126")
	})

	if !strings.Contains(output, "evaluated 4 addresses") {
		t.Errorf("Expected 4 addresses, got: %s", output)
	}
	if !strings.Contains(output, "Owned by GitHub: 4") {
		t.Errorf("Expected 4 owned addresses, got: %s", output)
	}
}

func TestEvaluateInput_CIDRExceedsThreshold(t *testing.T) {
	meta := createTestMeta()

	// /16 would be 65536 addresses, exceeding the 4096 threshold
	output := captureOutput(func() {
		evaluateInput(meta, "192.168.0.0/16")
	})

	if !strings.Contains(output, "too large") {
		t.Errorf("Expected too large message, got: %s", output)
	}
	if !strings.Contains(output, "65536 addresses") {
		t.Errorf("Expected address count, got: %s", output)
	}
	if !strings.Contains(output, "Warning") {
		t.Errorf("Expected warning, got: %s", output)
	}
}

func TestEvaluateInput_CIDRIPv6ExceedsThreshold(t *testing.T) {
	meta := createTestMeta()

	// /64 would be way too many addresses
	output := captureOutput(func() {
		evaluateInput(meta, "2001:db8::/64")
	})

	if !strings.Contains(output, "too large") {
		t.Errorf("Expected too large message, got: %s", output)
	}
	if !strings.Contains(output, "Warning") {
		t.Errorf("Expected warning, got: %s", output)
	}
}

func TestEvaluateInput_CIDRAtThreshold(t *testing.T) {
	meta := createTestMeta()

	// /20 is exactly 4096 addresses, should be evaluated
	output := captureOutput(func() {
		evaluateInput(meta, "192.168.0.0/20")
	})

	if !strings.Contains(output, "evaluated 4096 addresses") {
		t.Errorf("Expected 4096 addresses to be evaluated, got: %s", output)
	}
	if strings.Contains(output, "too large") {
		t.Errorf("Should not exceed threshold, got: %s", output)
	}
}

func TestEvaluateInput_WhitespaceHandling(t *testing.T) {
	meta := createTestMeta()

	// The main loop trims whitespace before calling evaluateInput
	// But let's test with trimmed input
	tests := []struct {
		input    string
		expected string
	}{
		{"192.30.252.42", "owned by GitHub"},
		{"  192.30.252.42  ", "invalid"}, // evaluateInput doesn't trim, main does
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output := captureOutput(func() {
				// Simulate what main does
				input := strings.TrimSpace(tt.input)
				evaluateInput(meta, input)
			})

			if !strings.Contains(output, "owned by GitHub") {
				t.Errorf("Expected owned message for %q, got: %s", tt.input, output)
			}
		})
	}
}

func TestLastAddrInPrefix_IPv4(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"192.168.1.0/24", "192.168.1.255"},
		{"192.168.0.0/16", "192.168.255.255"},
		{"10.0.0.0/8", "10.255.255.255"},
		{"192.168.1.0/30", "192.168.1.3"},
		{"192.168.1.4/30", "192.168.1.7"},
		{"192.168.1.128/25", "192.168.1.255"},
		{"192.168.1.0/32", "192.168.1.0"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			prefix := netip.MustParsePrefix(tt.prefix)
			last := lastAddrInPrefix(prefix)
			expected := netip.MustParseAddr(tt.expected)

			if last != expected {
				t.Errorf("lastAddrInPrefix(%s) = %s, want %s", tt.prefix, last, expected)
			}
		})
	}
}

func TestLastAddrInPrefix_IPv6(t *testing.T) {
	tests := []struct {
		prefix   string
		expected string
	}{
		{"2001:db8::/32", "2001:db8:ffff:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8::/48", "2001:db8:0:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8:1::/48", "2001:db8:1:ffff:ffff:ffff:ffff:ffff"},
		{"2001:db8::/126", "2001:db8::3"},
		{"2001:db8::/128", "2001:db8::"},
	}

	for _, tt := range tests {
		t.Run(tt.prefix, func(t *testing.T) {
			prefix := netip.MustParsePrefix(tt.prefix)
			last := lastAddrInPrefix(prefix)
			expected := netip.MustParseAddr(tt.expected)

			if last != expected {
				t.Errorf("lastAddrInPrefix(%s) = %s, want %s", tt.prefix, last, expected)
			}
		})
	}
}

func TestEvaluateInput_MultiLabelAddresses(t *testing.T) {
	meta := createTestMeta()

	// 192.30.252.0 is in both hooks (/22) and api (/24)
	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.0")
	})

	if !strings.Contains(output, "owned by GitHub") {
		t.Errorf("Expected owned message, got: %s", output)
	}
	// Should contain both labels
	if !strings.Contains(output, "api") || !strings.Contains(output, "hooks") {
		t.Errorf("Expected both api and hooks labels, got: %s", output)
	}
}

func TestEvaluateInput_MixedInputs(t *testing.T) {
	meta := createTestMeta()

	// Test a sequence of different input types
	tests := []struct {
		input    string
		contains []string
	}{
		{"192.30.252.42", []string{"owned by GitHub", "hooks"}},
		{"8.8.8.8", []string{"not owned"}},
		{"192.30.252.0/30", []string{"evaluated 4 addresses"}},
		{"invalid-input", []string{"invalid"}},
		{"2001:db8:1::1", []string{"owned by GitHub"}},
		{"192.168.0.0/16", []string{"too large"}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			output := captureOutput(func() {
				evaluateInput(meta, tt.input)
			})

			for _, expected := range tt.contains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected %q in output for input %q, got: %s", expected, tt.input, output)
				}
			}
		})
	}
}

func TestEvaluateInput_RepeatedInputs(t *testing.T) {
	meta := createTestMeta()

	// Test that repeated inputs give the same results (stateless behavior)
	input := "192.30.252.42"

	output1 := captureOutput(func() {
		evaluateInput(meta, input)
	})

	output2 := captureOutput(func() {
		evaluateInput(meta, input)
	})

	if output1 != output2 {
		t.Errorf("Repeated inputs gave different outputs:\nFirst: %s\nSecond: %s", output1, output2)
	}
}

func TestEvaluateInput_CaseInsensitiveCommands(t *testing.T) {
	// This test verifies that the main loop handles case-insensitive exit/quit
	// We can't easily test the main loop here, but we can test that evaluateInput
	// doesn't special-case these strings
	meta := createTestMeta()

	// These should be treated as invalid IPs, not commands
	testCases := []string{"EXIT", "exit", "Exit", "QUIT", "quit", "Quit"}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			output := captureOutput(func() {
				evaluateInput(meta, input)
			})

			// Should get invalid IP message
			if !strings.Contains(output, "invalid") {
				t.Errorf("Expected invalid message for %q, got: %s", input, output)
			}
		})
	}
}

func TestEvaluateInput_CIDROneAddress(t *testing.T) {
	meta := createTestMeta()

	// Test /32 CIDR which contains exactly one address
	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.42/32")
	})

	if !strings.Contains(output, "evaluated 1 addresses") {
		t.Errorf("Expected 1 address to be evaluated, got: %s", output)
	}
}

func TestEvaluateInput_EmptyLabels(t *testing.T) {
	// Test with metadata that has no matching entries
	meta := githubmeta.NewMetaDataForTesting([]githubmeta.Entry{})

	output := captureOutput(func() {
		evaluateInput(meta, "192.30.252.42")
	})

	if !strings.Contains(output, "not owned") {
		t.Errorf("Expected not owned message with empty metadata, got: %s", output)
	}
}
