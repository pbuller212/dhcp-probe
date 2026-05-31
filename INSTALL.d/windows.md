# Installing dhcp-probe on Windows

## Requirements

- Windows 10 / Windows Server 2016 or later (64-bit)
- [Npcap](https://npcap.com/) — the packet capture library for Windows

## Step 1 — Install Npcap

Download and run the Npcap installer from [https://npcap.com/#download](https://npcap.com/#download).

During installation, the default options are sufficient. You do **not** need "WinPcap compatibility mode".

## Step 2 — Download dhcp-probe

Download the latest `dhcp-probe-windows-amd64.exe` binary from the
[Releases](../../releases) page.

## Step 3 — Run as Administrator

`dhcp-probe` sends raw packets and requires Administrator privileges on Windows.

Open an **Administrator** Command Prompt or PowerShell and run:

```cmd
dhcp-probe-windows-amd64.exe [flags]
```

## Building from source

Building the Windows binary requires:

- Go 1.24+
- A C toolchain with Windows cross-compile support (MinGW-w64 or MSVC)
- The [Npcap SDK](https://npcap.com/dist/npcap-sdk-1.13.zip) (for header files and import libraries)

```cmd
set CGO_ENABLED=1
set CGO_CFLAGS=-I<path-to-npcap-sdk>\Include
set CGO_LDFLAGS=-L<path-to-npcap-sdk>\Lib\x64
go build -o dhcp-probe.exe .
```
