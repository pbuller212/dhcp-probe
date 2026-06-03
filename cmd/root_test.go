package cmd_test

import (
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/pbuller/dhcp-probe/cmd"
)

// TestMACFlag_Single verifies that a single valid MAC is parsed correctly.
func TestMACFlag_Single(t *testing.T) {
	macs, err := cmd.ParseMACs("aa:bb:cc:dd:ee:ff")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(macs) != 1 {
		t.Fatalf("got %d MACs, want 1", len(macs))
	}
	want, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")
	if macs[0].String() != want.String() {
		t.Errorf("got %v, want %v", macs[0], want)
	}
}

// TestMACFlag_CommaSeparated verifies that comma-separated MACs are all parsed.
func TestMACFlag_CommaSeparated(t *testing.T) {
	input := "aa:bb:cc:dd:ee:ff,00:11:22:33:44:55"
	macs, err := cmd.ParseMACs(input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(macs) != 2 {
		t.Fatalf("got %d MACs, want 2", len(macs))
	}
}

// TestMACFlag_Malformed verifies that a malformed MAC returns an error.
func TestMACFlag_Malformed(t *testing.T) {
	_, err := cmd.ParseMACs("not-a-mac")
	if err == nil {
		t.Error("expected error for malformed MAC, got nil")
	}
}

// TestMACFlag_MalformedInList verifies that a single bad MAC in a list fails the whole parse.
func TestMACFlag_MalformedInList(t *testing.T) {
	_, err := cmd.ParseMACs("aa:bb:cc:dd:ee:ff,bad")
	if err == nil {
		t.Error("expected error for malformed MAC in list, got nil")
	}
}

// TestVersionFlag verifies that --version causes ParseFlags to return ErrVersion.
func TestVersionFlag(t *testing.T) {
	_, err := cmd.ParseFlags([]string{"--version"})
	if err != cmd.ErrVersion {
		t.Errorf("ParseFlags(--version) error = %v, want ErrVersion", err)
	}
}

// TestVersionDefault verifies that the Version variable defaults to "dev".
func TestVersionDefault(t *testing.T) {
	if cmd.Version == "" {
		t.Error("Version is empty; want non-empty default")
	}
}

// TestTimeoutFlag verifies that the --timeout flag value is parsed to a time.Duration.
func TestTimeoutFlag(t *testing.T) {
	args := []string{"--timeout", "5s", "--interface", "lo"}
	cfg, err := cmd.ParseFlags(args)
	if err != nil {
		t.Fatalf("ParseFlags error: %v", err)
	}
	if cfg.Timeout != 5*time.Second {
		t.Errorf("Timeout = %v, want 5s", cfg.Timeout)
	}
}

// TestJSONOutput verifies that --json produces valid JSON containing all Offer fields.
func TestJSONOutput(t *testing.T) {
	// Use a stub runner so no network call is made.
	offers := []cmd.JSONOffer{
		{
			SourceMAC:  "00:11:22:33:44:55",
			ServerIP:   "192.168.1.1",
			ServerMAC:  "aa:bb:cc:dd:ee:ff",
			Manufacturer: "Test Vendor",
			OfferedIP:  "192.168.1.100",
			SubnetMask: "255.255.255.0",
			Gateway:    "192.168.1.1",
			DNS:        []string{"8.8.8.8", "8.8.4.4"},
			LeaseTime:  "24h0m0s",
		},
	}

	out, err := cmd.RenderJSON(offers)
	if err != nil {
		t.Fatalf("RenderJSON error: %v", err)
	}

	var decoded []map[string]any
	if err := json.Unmarshal([]byte(out), &decoded); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, out)
	}
	if len(decoded) != 1 {
		t.Fatalf("got %d entries, want 1", len(decoded))
	}

	requiredFields := []string{
		"source_mac", "server_ip", "server_mac", "manufacturer",
		"offered_ip", "subnet_mask", "gateway", "dns", "lease_time",
	}
	for _, field := range requiredFields {
		if _, ok := decoded[0][field]; !ok {
			t.Errorf("JSON output missing field %q", field)
		}
	}

}
