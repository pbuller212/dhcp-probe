package probe_test

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/pbuller/dhcp-probe/probe"
)

// rawDHCPOffer builds a minimal raw Ethernet+IP+UDP+DHCP OFFER frame.
//
// Layout:
//   - [0:6]   Dst MAC  (broadcast)
//   - [6:12]  Src MAC  (server MAC)
//   - [12:14] EtherType 0x0800 (IPv4)
//   - [14:34] IP header (20 bytes, no options)
//   - [34:42] UDP header (8 bytes)
//   - [42:]   DHCP payload (236 bytes fixed + options)
//
// The fixture encodes these values:
//
//	ServerMAC  = aa:bb:cc:dd:ee:ff
//	ServerIP   = 192.168.1.1
//	OfferedIP  = 192.168.1.100
//	SubnetMask = 255.255.255.0
//	Gateway    = 192.168.1.1
//	DNS        = [8.8.8.8, 8.8.4.4]
//	LeaseTime  = 86400s (24h)
//	SourceMAC  = 00:11:22:33:44:55 (client hardware address in DHCP header)
func rawDHCPOffer(t *testing.T) []byte {
	t.Helper()

	serverMAC := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	clientMAC := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	serverIP := net.ParseIP("192.168.1.1").To4()
	offeredIP := net.ParseIP("192.168.1.100").To4()

	// --- DHCP payload ---
	// Fixed-length portion: 236 bytes
	// op(1) htype(1) hlen(1) hops(1) xid(4) secs(2) flags(2)
	// ciaddr(4) yiaddr(4) siaddr(4) giaddr(4)
	// chaddr(16) sname(64) file(128)
	dhcp := make([]byte, 236)
	dhcp[0] = 2          // op = BOOTREPLY
	dhcp[1] = 1          // htype = Ethernet
	dhcp[2] = 6          // hlen = 6
	dhcp[3] = 0          // hops
	binary.BigEndian.PutUint32(dhcp[4:8], 0xdeadbeef) // xid
	// ciaddr = 0.0.0.0 (default)
	copy(dhcp[16:20], offeredIP)         // yiaddr
	copy(dhcp[20:24], serverIP)          // siaddr
	copy(dhcp[28:34], clientMAC)         // chaddr (first 6 bytes of 16-byte field)

	// DHCP options: magic cookie + options
	// Option 53: DHCP Message Type = DHCPOFFER (2)
	// Option 54: Server Identifier = 192.168.1.1
	// Option 51: IP Address Lease Time = 86400
	// Option 1:  Subnet Mask = 255.255.255.0
	// Option 3:  Router = 192.168.1.1
	// Option 6:  DNS = 8.8.8.8, 8.8.4.4
	// Option 255: End
	opts := []byte{
		99, 130, 83, 99, // magic cookie
		53, 1, 2,                   // DHCP Message Type: OFFER
		54, 4, 192, 168, 1, 1,      // Server Identifier
		51, 4, 0, 1, 81, 128,       // Lease Time: 86400 (0x00015180)
		1, 4, 255, 255, 255, 0,     // Subnet Mask
		3, 4, 192, 168, 1, 1,       // Router/Gateway
		6, 8, 8, 8, 8, 8, 8, 8, 4, 4, // DNS: 8.8.8.8 and 8.8.4.4
		255, // End
	}
	dhcpPayload := append(dhcp, opts...)

	// --- UDP header ---
	udpLen := uint16(8 + len(dhcpPayload))
	udp := make([]byte, 8)
	binary.BigEndian.PutUint16(udp[0:2], 67)     // src port (DHCP server)
	binary.BigEndian.PutUint16(udp[2:4], 68)     // dst port (DHCP client)
	binary.BigEndian.PutUint16(udp[4:6], udpLen) // length
	// checksum left as 0 (not validated in tests)

	// --- IP header ---
	ipTotalLen := uint16(20 + len(udp) + len(dhcpPayload))
	ip := make([]byte, 20)
	ip[0] = 0x45                                       // version=4, IHL=5
	binary.BigEndian.PutUint16(ip[2:4], ipTotalLen)
	ip[8] = 64                                         // TTL
	ip[9] = 17                                         // protocol = UDP
	copy(ip[12:16], serverIP)                          // src = server
	copy(ip[16:20], []byte{255, 255, 255, 255})        // dst = broadcast

	// --- Ethernet header ---
	eth := make([]byte, 14)
	copy(eth[0:6], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}) // dst = broadcast
	copy(eth[6:12], serverMAC)
	eth[12] = 0x08
	eth[13] = 0x00 // EtherType = IPv4

	frame := eth
	frame = append(frame, ip...)
	frame = append(frame, udp...)
	frame = append(frame, dhcpPayload...)
	return frame
}

// TestDecodeOffer_Fields verifies that DHCP options are correctly extracted.
func TestDecodeOffer_Fields(t *testing.T) {
	frame := rawDHCPOffer(t)
	offer, err := probe.DecodeOffer(frame)
	if err != nil {
		t.Fatalf("DecodeOffer returned error: %v", err)
	}

	if !offer.ServerIP.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("ServerIP = %v, want 192.168.1.1", offer.ServerIP)
	}
	if !offer.OfferedIP.Equal(net.ParseIP("192.168.1.100")) {
		t.Errorf("OfferedIP = %v, want 192.168.1.100", offer.OfferedIP)
	}
	ones, bits := offer.SubnetMask.Size()
	if ones != 24 || bits != 32 {
		t.Errorf("SubnetMask = %v, want /24", offer.SubnetMask)
	}
	if !offer.Gateway.Equal(net.ParseIP("192.168.1.1")) {
		t.Errorf("Gateway = %v, want 192.168.1.1", offer.Gateway)
	}
	if len(offer.DNS) != 2 {
		t.Fatalf("len(DNS) = %d, want 2", len(offer.DNS))
	}
	if !offer.DNS[0].Equal(net.ParseIP("8.8.8.8")) {
		t.Errorf("DNS[0] = %v, want 8.8.8.8", offer.DNS[0])
	}
	if !offer.DNS[1].Equal(net.ParseIP("8.8.4.4")) {
		t.Errorf("DNS[1] = %v, want 8.8.4.4", offer.DNS[1])
	}
	if offer.LeaseTime != 24*time.Hour {
		t.Errorf("LeaseTime = %v, want 24h", offer.LeaseTime)
	}
}

// TestDecodeOffer_ServerMAC verifies ServerMAC is taken from the Ethernet frame's src.
func TestDecodeOffer_ServerMAC(t *testing.T) {
	frame := rawDHCPOffer(t)
	offer, err := probe.DecodeOffer(frame)
	if err != nil {
		t.Fatalf("DecodeOffer returned error: %v", err)
	}

	want := net.HardwareAddr{0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}
	if offer.ServerMAC.String() != want.String() {
		t.Errorf("ServerMAC = %v, want %v", offer.ServerMAC, want)
	}
}

// TestDecodeOffer_SourceMAC verifies SourceMAC matches the client hardware address in the DHCP header.
func TestDecodeOffer_SourceMAC(t *testing.T) {
	frame := rawDHCPOffer(t)
	offer, err := probe.DecodeOffer(frame)
	if err != nil {
		t.Fatalf("DecodeOffer returned error: %v", err)
	}

	want := net.HardwareAddr{0x00, 0x11, 0x22, 0x33, 0x44, 0x55}
	if offer.SourceMAC.String() != want.String() {
		t.Errorf("SourceMAC = %v, want %v", offer.SourceMAC, want)
	}
}

// TestProbeWithTransport_AllSuccess_NoError verifies that a clean run returns nil error.
func TestProbeWithTransport_AllSuccess_NoError(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	transport := &probe.StubTransport{
		Frames: map[string][][]byte{
			mac1.String(): {buildOfferForMAC(t, mac1)},
			mac2.String(): {buildOfferForMAC(t, mac2)},
		},
	}

	offers, err := probe.ProbeWithTransport(transport, []net.HardwareAddr{mac1, mac2}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if len(offers) != 2 {
		t.Fatalf("got %d offers, want 2", len(offers))
	}
}

// TestProbeWithTransport_PartialError_ReturnsOffersAndError verifies that when one MAC fails,
// offers from successful MACs are still returned alongside a non-nil error.
func TestProbeWithTransport_PartialError_ReturnsOffersAndError(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	transport := &probe.StubTransport{
		Frames: map[string][][]byte{
			mac1.String(): {buildOfferForMAC(t, mac1)},
		},
		Errors: map[string]error{
			mac2.String(): fmt.Errorf("recv failed for %v", mac2),
		},
	}

	offers, err := probe.ProbeWithTransport(transport, []net.HardwareAddr{mac1, mac2}, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected non-nil error for partial failure, got nil")
	}
	if len(offers) != 1 {
		t.Fatalf("got %d offers, want 1", len(offers))
	}
	if !strings.Contains(err.Error(), mac2.String()) {
		t.Errorf("error %q does not mention failed MAC %v", err, mac2)
	}
}

// TestProbeWithTransport_AllErrors_ReturnsNoOffersAndError verifies that when all MACs fail,
// no offers are returned and the error is non-nil.
func TestProbeWithTransport_AllErrors_ReturnsNoOffersAndError(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	transport := &probe.StubTransport{
		Errors: map[string]error{
			mac1.String(): fmt.Errorf("recv failed for %v", mac1),
			mac2.String(): fmt.Errorf("recv failed for %v", mac2),
		},
	}

	offers, err := probe.ProbeWithTransport(transport, []net.HardwareAddr{mac1, mac2}, 100*time.Millisecond)
	if err == nil {
		t.Fatal("expected non-nil error when all MACs fail, got nil")
	}
	if len(offers) != 0 {
		t.Fatalf("got %d offers, want 0", len(offers))
	}
}

// TestProbe_ConcurrentMACs verifies that concurrent probing tags each result with its SourceMAC.
//
// This test uses a stub transport so no real network socket is opened.
func TestProbe_ConcurrentMACs(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")
	mac2, _ := net.ParseMAC("aa:bb:cc:dd:ee:ff")

	frame1 := rawDHCPOffer(t) // uses mac1 as chaddr
	// Build a second frame where chaddr is mac2.
	frame2 := buildOfferForMAC(t, mac2)

	transport := &probe.StubTransport{
		Frames: map[string][][]byte{
			mac1.String(): {frame1},
			mac2.String(): {frame2},
		},
	}

	offers, err := probe.ProbeWithTransport(transport, []net.HardwareAddr{mac1, mac2}, 100*time.Millisecond)
	if err != nil {
		t.Fatalf("ProbeWithTransport error: %v", err)
	}
	if len(offers) != 2 {
		t.Fatalf("got %d offers, want 2", len(offers))
	}

	byMAC := make(map[string]probe.Offer)
	for _, o := range offers {
		byMAC[o.SourceMAC.String()] = o
	}

	if _, ok := byMAC[mac1.String()]; !ok {
		t.Errorf("no offer tagged with SourceMAC %v", mac1)
	}
	if _, ok := byMAC[mac2.String()]; !ok {
		t.Errorf("no offer tagged with SourceMAC %v", mac2)
	}
}

// buildOfferForMAC creates a raw DHCPOFFER frame with the given MAC as the client hardware address.
func buildOfferForMAC(t *testing.T, mac net.HardwareAddr) []byte {
	t.Helper()

	frame := rawDHCPOffer(t)
	// chaddr is at Ethernet(14) + IP(20) + UDP(8) + DHCP fixed offset 28 = byte 70
	const chadrOffset = 14 + 20 + 8 + 28
	copy(frame[chadrOffset:chadrOffset+6], mac)
	return frame
}
