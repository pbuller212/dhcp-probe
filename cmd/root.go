package cmd

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net"
	"os"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/pbuller/dhcp-probe/probe"
)

// Config holds the parsed CLI flags.
type Config struct {
	Interface string
	Timeout   time.Duration
	MACs      []net.HardwareAddr
	JSON      bool
}

// JSONOffer is the serializable representation of a probe.Offer.
type JSONOffer struct {
	SourceMAC    string   `json:"source_mac"`
	ServerIP     string   `json:"server_ip"`
	ServerMAC    string   `json:"server_mac"`
	Manufacturer string   `json:"manufacturer"`
	OfferedIP    string   `json:"offered_ip"`
	SubnetMask   string   `json:"subnet_mask"`
	Gateway      string   `json:"gateway"`
	DNS          []string `json:"dns"`
	LeaseTime    string   `json:"lease_time"`
}

// ParseMACs parses a comma-separated list of MAC addresses.
func ParseMACs(s string) ([]net.HardwareAddr, error) {
	parts := strings.Split(s, ",")
	macs := make([]net.HardwareAddr, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		mac, err := net.ParseMAC(p)
		if err != nil {
			return nil, fmt.Errorf("invalid MAC %q: %w", p, err)
		}
		macs = append(macs, mac)
	}
	return macs, nil
}

// ParseFlags parses the given args slice and returns a Config.
func ParseFlags(args []string) (Config, error) {
	fs := flag.NewFlagSet("dhcp-probe", flag.ContinueOnError)

	iface := fs.String("interface", "", "network interface (default: first non-loopback)")
	timeout := fs.Duration("timeout", 3*time.Second, "wait window for DHCPOFFER responses")
	macStr := fs.String("mac", "", "comma-separated source MACs to probe with (default: interface MAC)")
	jsonOut := fs.Bool("json", false, "output as JSON")

	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}

	cfg := Config{
		Interface: *iface,
		Timeout:   *timeout,
		JSON:      *jsonOut,
	}

	if *macStr != "" {
		macs, err := ParseMACs(*macStr)
		if err != nil {
			return Config{}, err
		}
		cfg.MACs = macs
	}

	if cfg.Interface == "" {
		iface, err := defaultInterface()
		if err != nil {
			return Config{}, fmt.Errorf("no interface specified and could not detect one: %w", err)
		}
		cfg.Interface = iface
	}

	return cfg, nil
}

// RenderJSON marshals offers to indented JSON.
func RenderJSON(offers []JSONOffer) (string, error) {
	b, err := json.MarshalIndent(offers, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// renderTable writes offers as a tab-aligned table to stdout.
func renderTable(offers []JSONOffer) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "SOURCE MAC\tSERVER IP\tSERVER MAC\tMANUFACTURER\tOFFERED IP\tSUBNET MASK\tGATEWAY\tDNS\tLEASE")
	for _, o := range offers {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			o.SourceMAC,
			o.ServerIP,
			o.ServerMAC,
			o.Manufacturer,
			o.OfferedIP,
			o.SubnetMask,
			o.Gateway,
			strings.Join(o.DNS, " "),
			o.LeaseTime,
		)
	}
	w.Flush()
}

// toJSONOffer converts a probe.Offer to a JSONOffer.
func toJSONOffer(o probe.Offer) JSONOffer {
	dns := make([]string, len(o.DNS))
	for i, ip := range o.DNS {
		dns[i] = ip.String()
	}
	subnetMask := ""
	if o.SubnetMask != nil {
		ip := net.IP(o.SubnetMask)
		subnetMask = ip.String()
	}
	gateway := ""
	if o.Gateway != nil {
		gateway = o.Gateway.String()
	}
	return JSONOffer{
		SourceMAC:    o.SourceMAC.String(),
		ServerIP:     o.ServerIP.String(),
		ServerMAC:    o.ServerMAC.String(),
		Manufacturer: o.Manufacturer,
		OfferedIP:    o.OfferedIP.String(),
		SubnetMask:   subnetMask,
		Gateway:      gateway,
		DNS:          dns,
		LeaseTime:    o.LeaseTime.String(),
	}
}

// Execute is the CLI entry point.
func Execute() {
	cfg, err := ParseFlags(os.Args[1:])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}

	offers, err := probe.Probe(cfg.Interface, cfg.MACs, cfg.Timeout)
	if err != nil {
		if errors.Is(err, syscall.EPERM) || errors.Is(err, os.ErrPermission) {
			fmt.Fprintf(os.Stderr, "permission denied: raw socket requires root — try: sudo dhcp-probe\n")
		} else {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
		}
		os.Exit(1)
	}

	jsonOffers := make([]JSONOffer, len(offers))
	for i, o := range offers {
		jsonOffers[i] = toJSONOffer(o)
	}

	if cfg.JSON {
		out, err := RenderJSON(jsonOffers)
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: %v\n", err)
			os.Exit(1)
		}
		fmt.Println(out)
		return
	}

	renderTable(jsonOffers)
}

// defaultInterface returns the name of the first non-loopback network interface.
func defaultInterface() (string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return "", err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		if iface.Flags&net.FlagUp == 0 {
			continue
		}
		return iface.Name, nil
	}
	return "", fmt.Errorf("no suitable interface found")
}
