package githubmeta

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
)

const sampleMeta = `{
  "hooks": ["192.30.252.0/22", "2001:db8:1::/48"],
  "web": ["140.82.112.0/20"],
	"ssh_keys": ["ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOMqqnkVzrm0SdG6UOoqKLsabgH5C9okWi0dh2l9GKJl"],
  "verifiable_password_authentication": true,
  "ssh_key_fingerprints": {"SHA256": "example"}
}`

func TestParseMetaJSON(t *testing.T) {
	entries, err := parseMetaJSON(strings.NewReader(sampleMeta))
	if err != nil {
		t.Fatalf("parseMetaJSON returned error: %v", err)
	}

	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestLookup(t *testing.T) {
	entries, err := parseMetaJSON(strings.NewReader(sampleMeta))
	if err != nil {
		t.Fatalf("parseMetaJSON returned error: %v", err)
	}
	meta := newMetaData(entries)

	addr := netip.MustParseAddr("192.30.252.42")
	labels := meta.Lookup(addr)
	if len(labels) != 1 || labels[0] != "hooks" {
		t.Fatalf("expected [hooks], got %v", labels)
	}

	ipv6 := netip.MustParseAddr("2001:db8:1::213")
	labels = meta.Lookup(ipv6)
	if len(labels) != 1 || labels[0] != "hooks" {
		t.Fatalf("expected [hooks], got %v", labels)
	}

	unknown := netip.MustParseAddr("8.8.8.8")
	labels = meta.Lookup(unknown)
	if len(labels) != 0 {
		t.Fatalf("expected empty result, got %v", labels)
	}
}

func TestFetchWithCacheDir_UsesCacheOn304(t *testing.T) {
	tmpDir := t.TempDir()
	var calls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.Header().Set("ETag", `"v1"`)
			_, _ = w.Write([]byte(sampleMeta))
			return
		}
		if got := r.Header.Get("If-None-Match"); got != `"v1"` {
			t.Fatalf("expected If-None-Match header to be \"v1\", got %q", got)
		}
		w.WriteHeader(http.StatusNotModified)
	}))
	defer srv.Close()

	oldEndpoint := metaEndpoint
	metaEndpoint = srv.URL
	defer func() {
		metaEndpoint = oldEndpoint
	}()

	ctx := context.Background()
	client := srv.Client()

	first, err := FetchWithCacheDir(ctx, client, tmpDir)
	if err != nil {
		t.Fatalf("first fetch failed: %v", err)
	}
	if len(first.Entries()) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(first.Entries()))
	}

	second, err := FetchWithCacheDir(ctx, client, tmpDir)
	if err != nil {
		t.Fatalf("second fetch failed: %v", err)
	}
	if len(second.Entries()) != len(first.Entries()) {
		t.Fatalf("expected cached entries, got %d", len(second.Entries()))
	}
	if calls != 2 {
		t.Fatalf("expected 2 HTTP calls, got %d", calls)
	}
}

func TestFetchWithCacheDir_FallsBackToCacheOnError(t *testing.T) {
	tmpDir := t.TempDir()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", `"v1"`)
		_, _ = w.Write([]byte(sampleMeta))
	}))

	oldEndpoint := metaEndpoint
	metaEndpoint = srv.URL
	defer func() {
		metaEndpoint = oldEndpoint
	}()

	ctx := context.Background()
	client := srv.Client()

	if _, err := FetchWithCacheDir(ctx, client, tmpDir); err != nil {
		srv.Close()
		t.Fatalf("initial fetch failed: %v", err)
	}

	srv.Close()

	meta, err := FetchWithCacheDir(ctx, client, tmpDir)
	if err != nil {
		t.Fatalf("expected cached meta after error, got %v", err)
	}
	if len(meta.Entries()) != 3 {
		t.Fatalf("expected 3 cached entries, got %d", len(meta.Entries()))
	}
}
