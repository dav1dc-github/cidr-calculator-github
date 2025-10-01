package main

import (
	"bufio"
	"context"
	"fmt"
	"net/netip"
	"os"
	"strings"
	"time"

	"github.com/dav1dc-github/cidr-calculator-github/internal/githubmeta"
)

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
	addr, err := netip.ParseAddr(raw)
	if err != nil {
		fmt.Printf("%s -> invalid IP address (%v)\n", raw, err)
		return
	}

	labels := meta.Lookup(addr)
	if len(labels) == 0 {
		fmt.Printf("%s -> not owned by GitHub (based on current meta data)\n", addr)
		return
	}

	fmt.Printf("%s -> owned by GitHub (%s)\n", addr, strings.Join(labels, ", "))
}
