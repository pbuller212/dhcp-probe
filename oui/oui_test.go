package oui_test

import (
	"net"
	"testing"

	"github.com/pbuller/dhcp-probe/oui"
)

func TestLookupKnownOUI(t *testing.T) {
	mac, err := net.ParseMAC("28:6F:B9:01:02:03")
	if err != nil {
		t.Fatalf("parse MAC: %v", err)
	}
	result := oui.Lookup(mac)
	if result == "Unknown" {
		t.Errorf("expected a vendor name for 28:6F:B9, got 'Unknown'")
	}
}

func TestLookupUnknownMAC(t *testing.T) {
	// Locally-administered bit (bit 1 of first octet) guarantees this OUI is never in the IEEE registry.
	mac := net.HardwareAddr{0x02, 0x00, 0x00, 0x00, 0x00, 0x00}
	result := oui.Lookup(mac)
	if result != "Unknown" {
		t.Errorf("expected 'Unknown' for locally-administered MAC, got %q", result)
	}
}

func TestMapNonEmpty(t *testing.T) {
	count := oui.Count()
	if count == 0 {
		t.Error("OUI map is empty — CSV may not have loaded correctly")
	}
}
