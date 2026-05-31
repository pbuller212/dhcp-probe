# Changelog

All notable changes to this project will be documented in this file.

## [v1.0.0] - 2026-05-31

### Features

- **CLI** (`cmd/root.go`): Full CLI with `--interface`, `--timeout`, `--mac`, and `--json` flags; table and JSON output modes.
- **DHCP Probing** (`probe/`): Sends raw DHCPDISCOVER packets with random XID per probe; decodes DHCPOFFER responses into structured `Offer` values including server IP/MAC, offered IP, subnet mask, gateway, DNS servers, and lease time.
- **OUI Lookup** (`oui/`): Embedded IEEE MA-L OUI database for offline vendor/manufacturer resolution from MAC address prefixes; no network calls at runtime.
- **Windows support**: Npcap/gopacket transport layer for full support on Windows in addition to Linux.
- **Multi-MAC probing**: Probe with multiple source MACs in a single run to detect DHCP servers that respond differently per client.
- **CI/CD**: Forgejo Actions release workflow with cross-compiled binaries for Linux and Windows (amd64).

### Notes

- Requires `CAP_NET_RAW` capability (Linux) or Administrator/Npcap (Windows) to send raw packets.
- Go 1.24+ required.
