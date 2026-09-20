# dhcp-probe

A CLI tool to discover and report DHCP servers on the local network.

## Installation

There's no package manager distribution (no Homebrew/apt/Scoop/winget) — download a release
binary or build from source.

### Download

Download the latest binary for your platform from the
[Releases](https://github.com/pbuller212/dhcp-probe/releases) page.

Platform-specific install instructions (binary names to grab, post-download steps like
`chmod +x` / `setcap` / Npcap):

- [Linux](INSTALL.d/linux.md)
- [Windows](INSTALL.d/windows.md)

## Requirements

- Go 1.24+

## Build

```
git clone https://github.com/pbuller/dhcp-probe.git
cd dhcp-probe
go build -o dhcp-probe .
```

To install to `$GOPATH/bin`:

```
go install .
```

## Privileges

`dhcp-probe` sends raw packets and requires elevated privileges. Either run with `sudo`:

```
sudo dhcp-probe [flags]
```

Or grant the binary the `CAP_NET_RAW` capability so it can run without `sudo`:

```
sudo setcap cap_net_raw+ep ./dhcp-probe
./dhcp-probe [flags]
```

### Windows

On Windows, raw socket access requires running as **Administrator**. See [INSTALL.d/windows.md](INSTALL.d/windows.md) for full install and usage instructions.

## Usage

```
sudo dhcp-probe [flags]
```

| Flag | Default | Description |
|---|---|---|
| `--interface` | first non-loopback up interface | Network interface to probe on |
| `--timeout` | `3s` | Wait window for DHCPOFFER responses |
| `--mac` | interface MAC | Comma-separated source MACs to probe with |
| `--json` | `false` | Output as JSON instead of a table |
| `--version` | | Print version and exit |

### Table output (default)

Probe with multiple source MACs to detect DHCP servers that respond differently per client:

```
sudo dhcp-probe --mac aa:bb:cc:dd:ee:01,aa:bb:cc:dd:ee:02
```

```
SOURCE MAC         SERVER IP     SERVER MAC         MANUFACTURER      OFFERED IP     SUBNET MASK    GATEWAY       DNS           LEASE
aa:bb:cc:dd:ee:01  192.168.1.1   00:11:22:33:44:55  Cisco Systems     192.168.1.100  255.255.255.0  192.168.1.1   8.8.8.8       24h0m0s
aa:bb:cc:dd:ee:02  192.168.1.1   00:11:22:33:44:55  Cisco Systems     192.168.1.101  255.255.255.0  192.168.1.1   8.8.8.8       24h0m0s
aa:bb:cc:dd:ee:02  10.0.0.1      de:ad:be:ef:00:01  Unknown           10.0.0.50      255.255.255.0  10.0.0.1      1.1.1.1       12h0m0s
```

### JSON output

```
sudo dhcp-probe --json
```

```json
[
  {
    "source_mac": "aa:bb:cc:dd:ee:01",
    "server_ip": "192.168.1.1",
    "server_mac": "00:11:22:33:44:55",
    "manufacturer": "Cisco Systems",
    "offered_ip": "192.168.1.100",
    "subnet_mask": "255.255.255.0",
    "gateway": "192.168.1.1",
    "dns": ["8.8.8.8", "8.8.4.4"],
    "lease_time": "24h0m0s"
  },
  {
    "source_mac": "aa:bb:cc:dd:ee:02",
    "server_ip": "10.0.0.1",
    "server_mac": "de:ad:be:ef:00:01",
    "manufacturer": "Unknown",
    "offered_ip": "10.0.0.50",
    "subnet_mask": "255.255.255.0",
    "gateway": "10.0.0.1",
    "dns": ["1.1.1.1"],
    "lease_time": "12h0m0s"
  }
]
```

## Features

### OUI Lookup

The `oui` package resolves the vendor/organization name from a MAC address prefix using the IEEE MA-L (Organizationally Unique Identifier) registry, embedded at build time — no network calls at runtime.

```go
import "github.com/pbuller/dhcp-probe/oui"

name := oui.Lookup(mac) // e.g. "Nokia Shanghai Bell Co., Ltd." or "Unknown"
```

OUI data sourced from the [IEEE Public OUI Registry](https://standards-oui.ieee.org/oui/oui.csv).

## Dependencies

- [`github.com/insomniacslk/dhcp`](https://github.com/insomniacslk/dhcp) — DHCP client/server library
