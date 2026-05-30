// Package source fetches raw rule files over HTTP. Errors are always returned,
// never swallowed, so the generator can report which upstream sources failed
// rather than silently producing empty rule sets.
package source

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

var client = &http.Client{Timeout: 30 * time.Second}

// Fetch downloads url and returns its body.
func Fetch(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (sing-box-geosite generator)")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
