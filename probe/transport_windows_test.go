//go:build windows

package probe

import (
	"net"
	"testing"
)

func TestNewRawTransport_EmptyIface(t *testing.T) {
	_, err := newRawTransport("")
	if err == nil {
		t.Fatal("expected error for empty interface name, got nil")
	}
}

func TestNewRawTransport_InvalidIface(t *testing.T) {
	_, err := newRawTransport("no_such_iface_xyz")
	if err == nil {
		t.Fatal("expected error for nonexistent interface, got nil")
	}
}

// TestPcapTransport_ImplementsTransport verifies compile-time interface satisfaction.
var _ Transport = (*pcapTransport)(nil)

// TestSendDiscover_NilHandle verifies SendDiscover returns an error when the
// underlying pcap handle has been closed.
func TestSendDiscover_NilHandle(t *testing.T) {
	p := &pcapTransport{
		handle: nil,
		iface:  &net.Interface{HardwareAddr: make(net.HardwareAddr, 6)},
	}
	err := p.SendDiscover(make(net.HardwareAddr, 6), []byte{0x01, 0x02})
	if err == nil {
		t.Fatal("expected error sending on nil handle, got nil")
	}
}
