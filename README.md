# GitHub CIDR Lookup CLI

This Go-based CLI fetches the latest IP ranges published by GitHub and lets you check whether an IPv4 or IPv6 address (or entire CIDR ranges) belongs to any GitHub-operated network segment (for example `hooks`, `web`, `api`, and more).

## Prerequisites

- Go 1.22 or newer
- Internet access to reach `https://api.github.com/meta`

## Installation

### Install via `go install`

```sh
go install github.com/dav1dc-github/cidr-calculator-github@latest
```

This drops the `cidr-calculator-github` binary into `$(go env GOPATH)/bin` (or `GOBIN` if set); make sure that directory is on your `PATH`.

### Build from source

Clone the repository and download dependencies (none beyond the standard library are required):

```sh
git clone https://github.com/dav1dc-github/cidr-calculator-github.git
cd cidr-calculator-github
```

## Usage

The CLI accepts both single IP addresses and CIDR notation inputs.

### Single IP Address Lookups

Run the CLI with one or more IP addresses as arguments:

```sh
go run . 192.30.252.44 140.82.113.3 8.8.8.8
```

Example output:

```text
Fetching GitHub IP ranges...
Loaded 123 CIDR blocks from GitHub.
192.30.252.44 -> owned by GitHub (hooks)
140.82.113.3 -> owned by GitHub (web)
8.8.8.8 -> not owned by GitHub (based on current meta data)
```

### CIDR Range Evaluation

You can also provide CIDR notation to evaluate entire IP ranges. The tool will iterate through all addresses in the range (up to a threshold) and provide aggregated results:

```sh
go run . 192.30.252.0/30
```

Example output:

```text
Fetching GitHub IP ranges...
Loaded 123 CIDR blocks from GitHub.
192.30.252.0/30 -> evaluated 4 addresses:
  - Owned by GitHub: 4
  - Not owned: 0
  - Label distribution:
    - api,hooks: 4 addresses
```

For mixed ownership:

```sh
go run . 185.199.110.0/24
```

```text
185.199.110.0/24 -> evaluated 256 addresses:
  - Owned by GitHub: 256
  - Not owned: 0
  - Label distribution:
    - pages: 256 addresses
```

### IPv6 Support

Both IPv4 and IPv6 addresses and CIDR ranges are fully supported:

```sh
go run . 2001:db8:1::213 2001:db8:1::/126
```

Example output:

```text
2001:db8:1::213 -> owned by GitHub (hooks)
2001:db8:1::/126 -> evaluated 4 addresses:
  - Owned by GitHub: 4
  - Not owned: 0
  - Label distribution:
    - hooks: 4 addresses
```

### Threshold Behavior

To prevent excessive processing, CIDR ranges are limited to a maximum of **4,096 addresses** by default. Larger ranges will trigger a warning and skip evaluation:

```sh
go run . 192.168.0.0/16
```

```text
192.168.0.0/16 -> CIDR range too large (65536 addresses, threshold is 4096). Skipping evaluation.
Warning: Large CIDR ranges are not evaluated. Consider using a more specific range or adding a --limit flag in future versions.
```

Special cases:
- `/32` (IPv4) and `/128` (IPv6) prefixes contain exactly one address and are evaluated normally
- Ranges with exactly 4,096 addresses (e.g., `/20` for IPv4) are evaluated
- Ranges with more than 4,096 addresses are skipped with a warning

### Interactive Mode

If you call the binary without arguments it enters an interactive mode:

```sh
go run .
```

```text
Fetching GitHub IP ranges...
Loaded 123 CIDR blocks from GitHub.
Enter an IP address to check (type 'exit' to quit):
> 185.199.108.153
185.199.108.153 -> owned by GitHub (pages)
> 192.30.252.0/30
192.30.252.0/30 -> evaluated 4 addresses:
  - Owned by GitHub: 4
  - Not owned: 0
  - Label distribution:
    - api,hooks: 4 addresses
> exit
```

The interactive mode accepts:
- Single IP addresses (IPv4 or IPv6)
- CIDR notation (IPv4 or IPv6)
- `exit` or `quit` commands (case-insensitive) to terminate

Once installed via `go install`, you can run the compiled binary directly:

```sh
cidr-calculator-github 192.30.252.45
```

## Building a standalone binary

```sh
go build -o github-ip-checker
./github-ip-checker 140.82.114.4
```

## Testing

```sh
go test ./...
```

## Notes

- The tool queries the GitHub meta API on startup; subsequent lookups are performed locally without additional network calls.
- Matching is performed against the CIDR ranges published by GitHub. If an address is not listed, it may still belong to GitHub if their public ranges change between releases—rerun the CLI to refresh the data.
- Responses are cached under your OS cache directory (for example, `~/Library/Caches/cidr-calculator-github` on macOS). The CLI reuses cached metadata via the ETag header, reducing bandwidth while still refreshing when GitHub publishes new ranges. Delete the cache directory to force a full refetch.
