//go:build windows

package probe

import (
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/google/gopacket/pcap"
)

type pcapTransport struct {
	handle *pcap.Handle
	iface  *net.Interface
}

func newRawTransport(ifaceName string) (*pcapTransport, error) {
	ifi, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %q: %w", ifaceName, err)
	}

	// 100ms read timeout lets RecvOffers loop check its deadline without blocking forever.
	handle, err := pcap.OpenLive(ifaceName, 65535, true, 100*time.Millisecond)
	if err != nil {
		return nil, fmt.Errorf("pcap open %q: %w", ifaceName, err)
	}

	// Filter to only DHCP offer traffic (UDP dst port 68).
	if err := handle.SetBPFFilter("udp dst port 68"); err != nil {
		handle.Close()
		return nil, fmt.Errorf("bpf filter: %w", err)
	}

	return &pcapTransport{handle: handle, iface: ifi}, nil
}

func (p *pcapTransport) SendDiscover(_ net.HardwareAddr, pkt []byte) error {
	if p.handle == nil {
		return fmt.Errorf("pcap handle is nil")
	}
	frame := wrapFrame(p.iface.HardwareAddr, pkt)
	return p.handle.WritePacketData(frame)
}

func (p *pcapTransport) RecvOffers(_ net.HardwareAddr, timeout time.Duration) ([][]byte, error) {
	if p.handle == nil {
		return nil, fmt.Errorf("pcap handle is nil")
	}

	deadline := time.Now().Add(timeout)
	var frames [][]byte

	for time.Now().Before(deadline) {
		data, _, err := p.handle.ReadPacketData()
		if err != nil {
			// NextErrorTimeoutExpired from the 100ms read timeout — keep looping
			// until our own deadline is reached.
			if errors.Is(err, pcap.NextErrorTimeoutExpired) {
				continue
			}
			// Any other error: stop collecting and surface it.
			return nil, err
		}
		frame := make([]byte, len(data))
		copy(frame, data)
		frames = append(frames, frame)
	}
	return frames, nil
}

func (p *pcapTransport) close() {
	if p.handle != nil {
		p.handle.Close()
	}
}
