# dhcp-probe

A CLI tool to discover and report DHCP servers on the local network.

## Requirements

- Go 1.24+

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
