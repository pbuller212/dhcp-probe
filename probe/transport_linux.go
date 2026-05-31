//go:build linux

package probe

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/sys/unix"
)

type rawTransport struct {
	fd    int
	iface *net.Interface
}

func newRawTransport(ifaceName string) (*rawTransport, error) {
	ifi, err := net.InterfaceByName(ifaceName)
	if err != nil {
		return nil, fmt.Errorf("interface %q: %w", ifaceName, err)
	}

	// ETH_P_IP = 0x0800 in big-endian host order
	fd, err := unix.Socket(unix.AF_PACKET, unix.SOCK_RAW, int(htons(0x0800)))
	if err != nil {
		return nil, fmt.Errorf("raw socket: %w", err)
	}

	sa := &unix.SockaddrLinklayer{
		Protocol: htons(0x0800),
		Ifindex:  ifi.Index,
	}
	if err := unix.Bind(fd, sa); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("bind: %w", err)
	}

	return &rawTransport{fd: fd, iface: ifi}, nil
}

func (r *rawTransport) SendDiscover(_ net.HardwareAddr, pkt []byte) error {
	sa := &unix.SockaddrLinklayer{
		Protocol: htons(0x0800),
		Ifindex:  r.iface.Index,
		Halen:    6,
	}
	copy(sa.Addr[:6], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})

	// Wrap DHCP payload in Ethernet+IP+UDP before sending.
	frame := wrapFrame(r.iface.HardwareAddr, pkt)
	return unix.Sendto(r.fd, frame, 0, sa)
}

func (r *rawTransport) RecvOffers(_ net.HardwareAddr, timeout time.Duration) ([][]byte, error) {
	deadline := time.Now().Add(timeout)
	buf := make([]byte, 65535)
	var frames [][]byte

	for time.Now().Before(deadline) {
		tv := unix.NsecToTimeval(time.Until(deadline).Nanoseconds())
		if err := unix.SetsockoptTimeval(r.fd, unix.SOL_SOCKET, unix.SO_RCVTIMEO, &tv); err != nil {
			return nil, err
		}
		n, _, err := unix.Recvfrom(r.fd, buf, 0)
		if err != nil {
			break // timeout or error — stop collecting
		}
		frame := make([]byte, n)
		copy(frame, buf[:n])
		frames = append(frames, frame)
	}
	return frames, nil
}

func (r *rawTransport) close() {
	unix.Close(r.fd)
}
