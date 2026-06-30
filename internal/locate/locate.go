// Package locate discovers ndt7 test servers from the open M-Lab Locate API v2.
// It depends only on the standard library; no API key is required (FR-017).
package locate

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// DefaultURL is the M-Lab Locate v2 nearest-ndt7 endpoint.
const DefaultURL = "https://locate.measurementlab.net/v2/nearest/ndt/ndt7"

// Server is a candidate ndt7 test endpoint (data-model: Provider/Server).
type Server struct {
	Machine     string  `json:"machine"`
	City        string  `json:"city,omitempty"`
	Country     string  `json:"country,omitempty"`
	SiteCode    string  `json:"-"`
	Lat         float64 `json:"-"`
	Lon         float64 `json:"-"`
	HasCoords   bool    `json:"-"`
	DownloadURL string  `json:"-"`
	UploadURL   string  `json:"-"`
	IsFallback  bool    `json:"isFallback,omitempty"`
}

// Locator discovers candidate servers. Implemented by Client; faked in tests.
type Locator interface {
	Nearest(ctx context.Context) ([]Server, error)
}

// Client calls the M-Lab Locate API.
type Client struct {
	URL  string
	HTTP *http.Client
}

// NewClient returns a Locate client with sane defaults.
func NewClient() *Client {
	return &Client{
		URL:  DefaultURL,
		HTTP: &http.Client{Timeout: 10 * time.Second},
	}
}

// locateResponse mirrors the relevant subset of the Locate v2 payload.
type locateResponse struct {
	Results []struct {
		Machine  string `json:"machine"`
		Location struct {
			City    string `json:"city"`
			Country string `json:"country"`
		} `json:"location"`
		URLs map[string]string `json:"urls"`
	} `json:"results"`
}

// Nearest fetches proximity-ranked ndt7 servers. Coordinates are resolved from
// the bundled site table (Locate does not guarantee per-server lat/lon, U1).
func (c *Client) Nearest(ctx context.Context) ([]Server, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("build locate request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, fmt.Errorf("call locate api: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("locate api: unexpected status %d", resp.StatusCode)
	}

	var lr locateResponse
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("decode locate response: %w", err)
	}
	return parseServers(lr), nil
}

func parseServers(lr locateResponse) []Server {
	servers := make([]Server, 0, len(lr.Results))
	for _, r := range lr.Results {
		s := Server{
			Machine:     r.Machine,
			City:        r.Location.City,
			Country:     r.Location.Country,
			SiteCode:    SiteCode(r.Machine),
			DownloadURL: pickURL(r.URLs, "download"),
			UploadURL:   pickURL(r.URLs, "upload"),
		}
		if lat, lon, ok := Coords(s.SiteCode); ok {
			s.Lat, s.Lon, s.HasCoords = lat, lon, true
		}
		servers = append(servers, s)
	}
	return servers
}

// pickURL returns the first wss URL whose key contains the given test kind.
func pickURL(urls map[string]string, kind string) string {
	for k, v := range urls {
		if strings.HasPrefix(v, "wss://") && strings.Contains(k, kind) {
			return v
		}
	}
	return ""
}

// SiteCode extracts the M-Lab site code from a machine FQDN.
// "mlab1-gru01.mlab-oti.measurement-lab.org" -> "gru01".
func SiteCode(machine string) string {
	host := machine
	if i := strings.Index(host, "."); i >= 0 {
		host = host[:i] // mlab1-gru01
	}
	if i := strings.Index(host, "-"); i >= 0 {
		return host[i+1:] // gru01
	}
	return host
}
