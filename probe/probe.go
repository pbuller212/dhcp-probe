package probe

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/pbuller/dhcp-probe/oui"
)

// Offer holds the information extracted from a single DHCPOFFER response.
type Offer struct {
	SourceMAC    net.HardwareAddr // MAC used in outgoing DHCPDISCOVER (client chaddr)
	ServerIP     net.IP
	ServerMAC    net.HardwareAddr // Ethernet src MAC of the DHCPOFFER frame
	Manufacturer string
	OfferedIP    net.IP
	SubnetMask   net.IPMask
	Gateway      net.IP
	DNS          []net.IP
	LeaseTime    time.Duration
}

// Transport abstracts sending DHCP discover packets and receiving offer frames,
// allowing the real raw-socket implementation and test stubs to be swapped.
type Transport interface {
	SendDiscover(mac net.HardwareAddr, pkt []byte) error
	RecvOffers(mac net.HardwareAddr, timeout time.Duration) ([][]byte, error)
}

// StubTransport is a test double. Frames is keyed by mac.String().
// RecvOffers returns frames for the MAC, or the error from Errors if set.
type StubTransport struct {
	Frames map[string][][]byte
	Errors map[string]error
}

func (s *StubTransport) SendDiscover(_ net.HardwareAddr, _ []byte) error {
	return nil
}

func (s *StubTransport) RecvOffers(mac net.HardwareAddr, _ time.Duration) ([][]byte, error) {
	if s.Errors != nil {
		if err, ok := s.Errors[mac.String()]; ok {
			return nil, err
		}
	}
	return s.Frames[mac.String()], nil
}

// DecodeOffer parses a raw Ethernet+IP+UDP+DHCP frame and returns an Offer.
// It expects at least a full Ethernet (14) + IP (20) + UDP (8) + DHCP fixed (236) header.
func DecodeOffer(frame []byte) (Offer, error) {
	const (
		ethLen  = 14
		ipLen   = 20
		udpLen  = 8
		dhcpMin = 236
		minLen  = ethLen + ipLen + udpLen + dhcpMin
	)
	if len(frame) < minLen {
		return Offer{}, fmt.Errorf("frame too short: %d < %d", len(frame), minLen)
	}

	serverMAC := make(net.HardwareAddr, 6)
	copy(serverMAC, frame[6:12])

	dhcp := frame[ethLen+ipLen+udpLen:]

	offeredIP := make(net.IP, 4)
	copy(offeredIP, dhcp[16:20])

	serverIP := make(net.IP, 4)
	copy(serverIP, dhcp[20:24])

	sourceMAC := make(net.HardwareAddr, 6)
	copy(sourceMAC, dhcp[28:34])

	offer := Offer{
		SourceMAC: sourceMAC,
		ServerIP:  serverIP,
		ServerMAC: serverMAC,
		OfferedIP: offeredIP,
	}

	// Parse DHCP options after the magic cookie (4 bytes at offset 236).
	opts := dhcp[dhcpMin:]
	if len(opts) < 4 {
		return offer, nil
	}
	// Verify magic cookie 99.130.83.99
	if opts[0] != 99 || opts[1] != 130 || opts[2] != 83 || opts[3] != 99 {
		return offer, fmt.Errorf("invalid DHCP magic cookie")
	}
	opts = opts[4:]

	for i := 0; i < len(opts); {
		code := opts[i]
		if code == 255 { // End
			break
		}
		if code == 0 { // Pad
			i++
			continue
		}
		if i+1 >= len(opts) {
			break
		}
		length := int(opts[i+1])
		i += 2
		if i+length > len(opts) {
			break
		}
		val := opts[i : i+length]
		i += length

		switch code {
		case 1: // Subnet Mask
			if len(val) == 4 {
				offer.SubnetMask = net.IPMask(append([]byte(nil), val...))
			}
		case 3: // Router/Gateway
			if len(val) >= 4 {
				offer.Gateway = net.IP(append([]byte(nil), val[:4]...))
			}
		case 6: // DNS
			for j := 0; j+4 <= len(val); j += 4 {
				offer.DNS = append(offer.DNS, net.IP(append([]byte(nil), val[j:j+4]...)))
			}
		case 51: // Lease Time
			if len(val) == 4 {
				secs := binary.BigEndian.Uint32(val)
				offer.LeaseTime = time.Duration(secs) * time.Second
			}
		case 54: // Server Identifier — overrides siaddr if present
			if len(val) == 4 {
				offer.ServerIP = net.IP(append([]byte(nil), val...))
			}
		}
	}

	offer.Manufacturer = oui.Lookup(offer.ServerMAC)
	return offer, nil
}

// ProbeWithTransport runs one goroutine per MAC, sends a DHCPDISCOVER via t,
// collects DHCPOFFER frames, and returns the merged slice of Offers.
func ProbeWithTransport(t Transport, macs []net.HardwareAddr, timeout time.Duration) ([]Offer, error) {
	type result struct {
		offers []Offer
		err    error
	}

	ch := make(chan result, len(macs))
	var wg sync.WaitGroup

	for _, mac := range macs {
		mac := mac
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := t.SendDiscover(mac, buildDiscover(mac)); err != nil {
				ch <- result{err: err}
				return
			}
			frames, err := t.RecvOffers(mac, timeout)
			if err != nil {
				ch <- result{err: err}
				return
			}
			var offers []Offer
			for _, f := range frames {
				o, err := DecodeOffer(f)
				if err != nil {
					continue
				}
				offers = append(offers, o)
			}
			ch <- result{offers: offers}
		}()
	}

	wg.Wait()
	close(ch)

	var all []Offer
	var errs []error
	for r := range ch {
		if r.err != nil {
			errs = append(errs, r.err)
			continue
		}
		all = append(all, r.offers...)
	}
	return all, errors.Join(errs...)
}

// Probe is the real entry point. It opens a raw socket on iface, sends
// DHCPDISCOVER packets for each MAC, and returns all DHCPOFFER responses.
// If macs is empty, the real MAC address of iface is used.
func Probe(iface string, macs []net.HardwareAddr, timeout time.Duration) ([]Offer, error) {
	if len(macs) == 0 {
		ifi, err := net.InterfaceByName(iface)
		if err != nil {
			return nil, fmt.Errorf("interface %q not found: %w", iface, err)
		}
		macs = []net.HardwareAddr{ifi.HardwareAddr}
	}
	for _, mac := range macs {
		if _, err := net.ParseMAC(mac.String()); err != nil {
			return nil, fmt.Errorf("invalid MAC %v: %w", mac, err)
		}
	}
	t, err := newRawTransport(iface)
	if err != nil {
		return nil, err
	}
	defer t.close()
	return ProbeWithTransport(t, macs, timeout)
}

// buildDiscover constructs a minimal DHCPDISCOVER packet (Ethernet+IP+UDP+DHCP).
// The returned bytes are suitable for writing to a raw socket.
func buildDiscover(mac net.HardwareAddr) []byte {
	// DHCP fixed header: 236 bytes
	dhcp := make([]byte, 236)
	dhcp[0] = 1 // op = BOOTREQUEST
	dhcp[1] = 1 // htype = Ethernet
	dhcp[2] = 6 // hlen
	var xid [4]byte
	_, _ = rand.Read(xid[:])
	copy(dhcp[4:8], xid[:])
	dhcp[10] = 0x80 // flags: broadcast
	copy(dhcp[28:34], mac)

	opts := []byte{
		99, 130, 83, 99, // magic cookie
		53, 1, 1, // DHCP Message Type: DISCOVER
		255,       // End
	}
	return append(dhcp, opts...)
}
