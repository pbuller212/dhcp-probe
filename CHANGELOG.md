# Changelog

All notable changes to this project will be documented in this file.

## [Unreleased]

### CI/CD

- **GitHub release mirroring** (`.forgejo/workflows/release.yml`): New `mirror-github-release` job runs after a tagged build's Forgejo release is published. It creates (or reuses, by tag lookup) a matching GitHub Release on `pbuller212/dhcp-probe` via the GitHub REST API and uploads the same cross-compiled binaries as release assets, skipping any that were already uploaded. Requires a fine-grained GitHub PAT (Releases: write, scoped to this repo) stored as the Forgejo Actions secret `GH_RELEASE_TOKEN`. Forgejo Actions remains the only build pipeline; GitHub receives finished artifacts only.

### Bug Fixes

- **RecvOffers captured unrelated frames** (`probe/probe.go`): `ProbeWithTransport` now discards frames that are not valid DHCPOFFER responses for the target MAC before calling `DecodeOffer`. The new `MatchesOffer` helper checks UDP destination port 68, the DHCP magic cookie, `chaddr` equality, and DHCP option 53 == 2 (OFFER). Previously all IP frames on the interface were passed to `DecodeOffer`, which could return spurious offers under network load.

- **Partial probe failure** (`probe/probe.go`, `cmd/root.go`): `ProbeWithTransport` no longer discards all results when one goroutine errors. Per-MAC errors are now accumulated with `errors.Join` and returned alongside any successfully collected offers. The CLI prints partial results with a stderr warning and exits with code `2` on partial failure; it exits `1` only when all probes fail.

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
