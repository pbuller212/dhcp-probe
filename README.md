# dhcp-probe

A CLI tool to discover and report DHCP servers on the local network.

## Requirements

- Go 1.24+

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

Raw socket access requires root privileges.

### Table output (default)

```
SOURCE MAC         SERVER IP     SERVER MAC         MANUFACTURER  OFFERED IP     SUBNET MASK    GATEWAY       DNS           LEASE
aa:bb:cc:dd:ee:ff  192.168.1.1   00:11:22:33:44:55  Acme Corp     192.168.1.100  255.255.255.0  192.168.1.1   8.8.8.8       24h0m0s
```

### JSON output

```
sudo dhcp-probe --json
```

```json
[
  {
    "source_mac": "aa:bb:cc:dd:ee:ff",
    "server_ip": "192.168.1.1",
    "server_mac": "00:11:22:33:44:55",
    "manufacturer": "Acme Corp",
    "offered_ip": "192.168.1.100",
    "subnet_mask": "255.255.255.0",
    "gateway": "192.168.1.1",
    "dns": ["8.8.8.8", "8.8.4.4"],
    "lease_time": "24h0m0s"
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
