package itunes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type AppResult struct {
	BundleID  string  `json:"bundle_id"`
	Name      string  `json:"name"`
	IconURL   string  `json:"icon_url"`
	Version   string  `json:"version"`
	Seller    string  `json:"seller"`
	Price     float64 `json:"price"`
	AppID     int64   `json:"app_id"`
	SizeBytes int64   `json:"size_bytes"`
}

type lookupResponse struct {
	ResultCount int `json:"resultCount"`
	Results     []struct {
		TrackID       int64     `json:"trackId"`
		TrackName     string    `json:"trackName"`
		BundleID      string    `json:"bundleId"`
		ArtworkURL100 string    `json:"artworkUrl100"`
		Version       string    `json:"version"`
		SellerName    string    `json:"sellerName"`
		Price         float64   `json:"price"`
		FileSizeBytes flexInt64 `json:"fileSizeBytes"`
	} `json:"results"`
}

// flexInt64 accepts JSON numbers or numeric strings (iTunes search returns
// fileSizeBytes as a string).
type flexInt64 int64

func (n *flexInt64) UnmarshalJSON(b []byte) error {
	if string(b) == "null" {
		*n = 0
		return nil
	}
	if len(b) > 0 && b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		s = strings.TrimSpace(s)
		if s == "" {
			*n = 0
			return nil
		}
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return err
		}
		*n = flexInt64(v)
		return nil
	}
	var v int64
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*n = flexInt64(v)
	return nil
}

var client = &http.Client{Timeout: 15 * time.Second}

func Lookup(bundleID, country string) ([]AppResult, error) {
	country = strings.ToLower(strings.TrimSpace(country))
	if country == "" {
		country = "us"
	}
	bundleID = strings.TrimSpace(bundleID)
	if bundleID == "" {
		return nil, fmt.Errorf("bundle id required")
	}
	u := fmt.Sprintf("https://itunes.apple.com/lookup?bundleId=%s&country=%s", url.QueryEscape(bundleID), url.QueryEscape(country))
	return fetch(u)
}

func Search(term, country string, limit int) ([]AppResult, error) {
	country = strings.ToLower(strings.TrimSpace(country))
	if country == "" {
		country = "us"
	}
	term = strings.TrimSpace(term)
	if term == "" {
		return nil, fmt.Errorf("search term required")
	}
	if limit <= 0 {
		limit = 15
	}
	u := fmt.Sprintf("https://itunes.apple.com/search?term=%s&entity=software&country=%s&limit=%d",
		url.QueryEscape(term), url.QueryEscape(country), limit)
	return fetch(u)
}

func fetch(u string) ([]AppResult, error) {
	req, err := http.NewRequest(http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return nil, fmt.Errorf("itunes status %d", resp.StatusCode)
	}
	var raw lookupResponse
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, err
	}
	out := make([]AppResult, 0, len(raw.Results))
	for _, r := range raw.Results {
		icon := r.ArtworkURL100
		if icon != "" {
			icon = strings.Replace(icon, "100x100bb", "512x512bb", 1)
		}
		out = append(out, AppResult{
			BundleID:  r.BundleID,
			Name:      r.TrackName,
			IconURL:   icon,
			Version:   r.Version,
			Seller:    r.SellerName,
			Price:     r.Price,
			AppID:     r.TrackID,
			SizeBytes: int64(r.FileSizeBytes),
		})
	}
	return out, nil
}
