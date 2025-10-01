# GitHub CIDR Lookup CLI

This Go-based CLI fetches the latest IP ranges published by GitHub and lets you check whether an IPv4 or IPv6 address belongs to any GitHub-operated network segment (for example `hooks`, `web`, `api`, and more).

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
> exit
```

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
