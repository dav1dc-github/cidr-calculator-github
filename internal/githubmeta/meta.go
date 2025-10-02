package githubmeta

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/netip"
	"os"
	"path/filepath"
	"sort"
	"time"
)

const metaURL = "https://api.github.com/meta"

var metaEndpoint = metaURL

// Entry describes a single CIDR block tagged with the GitHub subsystem it belongs to.
type Entry struct {
	Label  string
	Prefix netip.Prefix
}

// MetaData contains all CIDR entries from the GitHub meta endpoint and offers lookup utilities.
type MetaData struct {
	entries []Entry
}

// Fetch downloads the GitHub meta endpoint and parses the CIDR information.
func Fetch(ctx context.Context, client *http.Client) (*MetaData, error) {
	cacheDir, err := defaultCacheDir()
	if err != nil {
		return fetch(ctx, client, nil)
	}

	return fetch(ctx, client, newCacheStore(cacheDir))
}

// FetchWithCacheDir downloads the GitHub meta endpoint using a user-provided cache directory.
// An empty cacheDir disables on-disk caching.
func FetchWithCacheDir(ctx context.Context, client *http.Client, cacheDir string) (*MetaData, error) {
	return fetch(ctx, client, newCacheStore(cacheDir))
}

// parseMetaJSON converts the JSON response into a slice of entries.
func parseMetaJSON(r io.Reader) ([]Entry, error) {
	var raw map[string]any
	if err := json.NewDecoder(r).Decode(&raw); err != nil {
		return nil, fmt.Errorf("decode meta response: %w", err)
	}

	var entries []Entry
	for label, value := range raw {
		cidrs, ok := extractStringSlice(value)
		if !ok {
			continue
		}
		for _, cidr := range cidrs {
			prefix, err := netip.ParsePrefix(cidr)
			if err != nil {
				continue
			}
			entries = append(entries, Entry{Label: label, Prefix: prefix})
		}
	}

	if len(entries) == 0 {
		return nil, errors.New("no CIDR entries found in meta response")
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Label == entries[j].Label {
			return entries[i].Prefix.String() < entries[j].Prefix.String()
		}
		return entries[i].Label < entries[j].Label
	})

	return entries, nil
}

func extractStringSlice(value any) ([]string, bool) {
	rawSlice, ok := value.([]any)
	if !ok {
		return nil, false
	}

	out := make([]string, 0, len(rawSlice))
	for _, item := range rawSlice {
		str, ok := item.(string)
		if !ok {
			return nil, false
		}
		out = append(out, str)
	}
	return out, true
}

func newMetaData(entries []Entry) *MetaData {
	copyEntries := make([]Entry, len(entries))
	copy(copyEntries, entries)
	return &MetaData{entries: copyEntries}
}

// NewMetaDataForTesting creates a MetaData instance for testing purposes.
// This should only be used in tests.
func NewMetaDataForTesting(entries []Entry) *MetaData {
	return newMetaData(entries)
}

// Entries exposes a copy of the parsed entries.
func (m *MetaData) Entries() []Entry {
	if m == nil {
		return nil
	}
	out := make([]Entry, len(m.entries))
	copy(out, m.entries)
	return out
}

// Lookup returns the GitHub subsystems whose CIDR ranges contain the provided IP address.
func (m *MetaData) Lookup(addr netip.Addr) []string {
	if m == nil || !addr.IsValid() {
		return nil
	}

	labels := make([]string, 0, 2)
	seen := make(map[string]struct{})
	for _, entry := range m.entries {
		if entry.Prefix.Contains(addr) {
			if _, exists := seen[entry.Label]; !exists {
				labels = append(labels, entry.Label)
				seen[entry.Label] = struct{}{}
			}
		}
	}

	sort.Strings(labels)
	return labels
}

// FetchWithTimeout is a convenience helper that applies a timeout to the fetch operation.
func FetchWithTimeout(timeout time.Duration) (*MetaData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return Fetch(ctx, http.DefaultClient)
}

func fetch(ctx context.Context, client *http.Client, store *cacheStore) (*MetaData, error) {
	if client == nil {
		client = http.DefaultClient
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, metaEndpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "cidr-calculator-github/1.0")

	if etag := store.readETag(); etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := client.Do(req)
	if err != nil {
		if meta, cacheErr := store.load(); cacheErr == nil {
			return meta, nil
		}
		return nil, fmt.Errorf("fetch github meta: %w", err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusNotModified:
		meta, err := store.load()
		if err != nil {
			return nil, fmt.Errorf("load cached meta after 304: %w", err)
		}
		return meta, nil
	case http.StatusOK:
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("read meta response: %w", err)
		}
		entries, err := parseMetaJSON(bytes.NewReader(raw))
		if err != nil {
			return nil, err
		}
		if err := store.save(raw, resp.Header.Get("ETag")); err != nil {
			// caching failures are non-fatal
		}
		return newMetaData(entries), nil
	default:
		if meta, cacheErr := store.load(); cacheErr == nil {
			return meta, nil
		}
		return nil, fmt.Errorf("unexpected status %d from meta endpoint", resp.StatusCode)
	}
}

func defaultCacheDir() (string, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "cidr-calculator-github"), nil
}

type cacheStore struct {
	dir string
}

func newCacheStore(dir string) *cacheStore {
	if dir == "" {
		return nil
	}
	return &cacheStore{dir: dir}
}

func (c *cacheStore) metaPath() string {
	return filepath.Join(c.dir, "meta.json")
}

func (c *cacheStore) etagPath() string {
	return filepath.Join(c.dir, "meta.etag")
}

func (c *cacheStore) readETag() string {
	if c == nil {
		return ""
	}
	data, err := os.ReadFile(c.etagPath())
	if err != nil {
		return ""
	}
	return string(bytes.TrimSpace(data))
}

func (c *cacheStore) load() (*MetaData, error) {
	if c == nil {
		return nil, errors.New("cache disabled")
	}
	raw, err := os.ReadFile(c.metaPath())
	if err != nil {
		return nil, err
	}
	entries, err := parseMetaJSON(bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	return newMetaData(entries), nil
}

func (c *cacheStore) save(raw []byte, etag string) error {
	if c == nil {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(c.metaPath(), raw, 0o644); err != nil {
		return err
	}
	if etag != "" {
		if err := os.WriteFile(c.etagPath(), []byte(etag), 0o644); err != nil {
			return err
		}
	}
	return nil
}
