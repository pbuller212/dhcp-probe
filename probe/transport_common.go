package probe

import "net"

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
	ip[8] = 64 // TTL
	ip[9] = 17 // UDP
	copy(ip[12:16], []byte{0, 0, 0, 0})
	copy(ip[16:20], []byte{255, 255, 255, 255})
	putU16(ip[10:], ipChecksum(ip))

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

// ipChecksum computes the RFC-791 one's-complement checksum over a 20-byte IPv4 header.
// The checksum field (bytes 10-11) must be zero when this is called.
func ipChecksum(hdr []byte) uint16 {
	var sum uint32
	for i := 0; i < len(hdr); i += 2 {
		sum += uint32(hdr[i])<<8 | uint32(hdr[i+1])
	}
	for sum>>16 != 0 {
		sum = (sum & 0xffff) + (sum >> 16)
	}
	return ^uint16(sum)
}
