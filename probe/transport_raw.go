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

// htons converts a uint16 from host to network byte order.
func htons(v uint16) uint16 {
	return (v<<8)&0xff00 | v>>8
}

// wrapFrame wraps a DHCP payload in a broadcast Ethernet+IP+UDP frame.
func wrapFrame(srcMAC net.HardwareAddr, dhcpPayload []byte) []byte {
	udpLen := uint16(8 + len(dhcpPayload))
	ipLen := uint16(20) + udpLen

	eth := make([]byte, 14)
	copy(eth[0:6], []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff})
	copy(eth[6:12], srcMAC)
	eth[12], eth[13] = 0x08, 0x00

	ip := make([]byte, 20)
	ip[0] = 0x45
	putU16(ip[2:], ipLen)
	ip[8] = 64  // TTL
	ip[9] = 17  // UDP
	copy(ip[12:16], []byte{0, 0, 0, 0})
	copy(ip[16:20], []byte{255, 255, 255, 255})

	udp := make([]byte, 8)
	putU16(udp[0:], 68) // src port: DHCP client
	putU16(udp[2:], 67) // dst port: DHCP server
	putU16(udp[4:], udpLen)

	frame := eth
	frame = append(frame, ip...)
	frame = append(frame, udp...)
	frame = append(frame, dhcpPayload...)
	return frame
}

func putU16(b []byte, v uint16) {
	b[0] = byte(v >> 8)
	b[1] = byte(v)
}
