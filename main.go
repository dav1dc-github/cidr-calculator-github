package main

import (
	"bufio"
	"context"
	"fmt"
	"net/netip"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/dav1dc-github/cidr-calculator-github/internal/githubmeta"
)

const defaultThreshold = 4096

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	fmt.Println("Fetching GitHub IP ranges...")
	meta, err := githubmeta.Fetch(ctx, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Loaded %d CIDR blocks from GitHub.\n", len(meta.Entries()))

	args := os.Args[1:]
	if len(args) > 0 {
		for _, arg := range args {
			evaluateInput(meta, arg)
		}
		return
	}

	fmt.Println("Enter an IP address to check (type 'exit' to quit):")
	scanner := bufio.NewScanner(os.Stdin)
	for {
		fmt.Print("> ")
		if !scanner.Scan() {
			if err := scanner.Err(); err != nil {
				fmt.Fprintf(os.Stderr, "input error: %v\n", err)
			}
			break
		}
		input := strings.TrimSpace(scanner.Text())
		if input == "" {
			continue
		}
		if strings.EqualFold(input, "exit") || strings.EqualFold(input, "quit") {
			break
		}
		evaluateInput(meta, input)
	}
}

func evaluateInput(meta *githubmeta.MetaData, raw string) {
	// Try parsing as single IP address first
	addr, err := netip.ParseAddr(raw)
	if err == nil {
		evaluateAddr(meta, raw, addr)
		return
	}

	// Try parsing as CIDR prefix
	prefix, err := netip.ParsePrefix(raw)
	if err != nil {
		fmt.Printf("%s -> invalid IP address or CIDR (%v)\n", raw, err)
		return
	}

	evaluateCIDR(meta, raw, prefix)
}

func evaluateAddr(meta *githubmeta.MetaData, raw string, addr netip.Addr) {
	labels := meta.Lookup(addr)
	if len(labels) == 0 {
		fmt.Printf("%s -> not owned by GitHub (based on current meta data)\n", raw)
		return
	}

	fmt.Printf("%s -> owned by GitHub (%s)\n", raw, strings.Join(labels, ", "))
}

func evaluateCIDR(meta *githubmeta.MetaData, raw string, prefix netip.Prefix) {
	// Calculate the number of addresses in the prefix
	bits := prefix.Bits()
	addrBits := 32
	if prefix.Addr().Is6() {
		addrBits = 128
	}
	hostBits := addrBits - bits

	// Check if the range is too large
	var count uint64
	if hostBits >= 64 {
		// Would overflow uint64, definitely over threshold
		count = defaultThreshold + 1
	} else {
		count = 1 << hostBits
	}

	if count > defaultThreshold {
		fmt.Printf("%s -> CIDR range too large (%d addresses, threshold is %d). Skipping evaluation.\n", 
			raw, count, defaultThreshold)
		fmt.Printf("Warning: Large CIDR ranges are not evaluated. Consider using a more specific range or adding a --limit flag in future versions.\n")
		return
	}

	// Iterate through all addresses in the CIDR range
	ownedCount := 0
	nonOwnedCount := 0
	labelSets := make(map[string]int) // label set signature -> count
	
	addr := prefix.Addr()
	lastAddr := lastAddrInPrefix(prefix)
	
	for {
		labels := meta.Lookup(addr)
		if len(labels) == 0 {
			nonOwnedCount++
		} else {
			ownedCount++
			// Create a signature for this set of labels
			sort.Strings(labels)
			sig := strings.Join(labels, ",")
			labelSets[sig]++
		}

		if addr == lastAddr {
			break
		}
		addr = addr.Next()
	}

	// Print summary
	totalCount := ownedCount + nonOwnedCount
	fmt.Printf("%s -> evaluated %d addresses:\n", raw, totalCount)
	fmt.Printf("  - Owned by GitHub: %d\n", ownedCount)
	fmt.Printf("  - Not owned: %d\n", nonOwnedCount)
	
	if len(labelSets) > 0 {
		fmt.Printf("  - Label distribution:\n")
		// Sort label sets for consistent output
		var sigs []string
		for sig := range labelSets {
			sigs = append(sigs, sig)
		}
		sort.Strings(sigs)
		
		for _, sig := range sigs {
			fmt.Printf("    - %s: %d addresses\n", sig, labelSets[sig])
		}
	}
}

// lastAddrInPrefix returns the last IP address in the given prefix
func lastAddrInPrefix(prefix netip.Prefix) netip.Addr {
	addr := prefix.Addr()
	bits := prefix.Bits()
	
	if addr.Is4() {
		a := addr.As4()
		
		// Set all host bits to 1
		for i := 0; i < 4; i++ {
			byteStart := i * 8
			byteEnd := byteStart + 8
			
			if byteEnd <= bits {
				// All bits in this byte are network bits, keep as is
				continue
			} else if byteStart >= bits {
				// All bits in this byte are host bits
				a[i] = 0xff
			} else {
				// Mixed: some network, some host
				hostBitsInByte := byteEnd - bits
				mask := byte((1 << hostBitsInByte) - 1)
				a[i] |= mask
			}
		}
		
		return netip.AddrFrom4(a)
	} else {
		a := addr.As16()
		
		// Set all host bits to 1
		for i := 0; i < 16; i++ {
			byteStart := i * 8
			byteEnd := byteStart + 8
			
			if byteEnd <= bits {
				// All bits in this byte are network bits, keep as is
				continue
			} else if byteStart >= bits {
				// All bits in this byte are host bits
				a[i] = 0xff
			} else {
				// Mixed: some network, some host
				hostBitsInByte := byteEnd - bits
				mask := byte((1 << hostBitsInByte) - 1)
				a[i] |= mask
			}
		}
		
		return netip.AddrFrom16(a)
	}
}
