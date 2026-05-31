# Installing dhcp-probe on Linux

## Requirements

- Linux kernel with `AF_PACKET` raw socket support (standard on all modern distributions)
- Go 1.24+ (for building from source)

## Build

```sh
git clone https://github.com/pbuller/dhcp-probe.git
cd dhcp-probe
go build -o dhcp-probe .
```

## Privileges

`dhcp-probe` uses raw packet sockets and requires elevated privileges. Two options:

**Option 1 — run with sudo:**

```sh
sudo ./dhcp-probe [flags]
```

**Option 2 — grant `CAP_NET_RAW` (run without sudo):**

```sh
sudo setcap cap_net_raw+ep ./dhcp-probe
./dhcp-probe [flags]
```

The `setcap` approach is preferred for repeated use; it grants only the minimum capability required.

## Pre-built binaries

Download the latest `dhcp-probe-linux-amd64` or `dhcp-probe-linux-arm64` binary from the
[Releases](../../releases) page. Make it executable:

```sh
chmod +x dhcp-probe-linux-amd64
sudo setcap cap_net_raw+ep ./dhcp-probe-linux-amd64
./dhcp-probe-linux-amd64 [flags]
```
