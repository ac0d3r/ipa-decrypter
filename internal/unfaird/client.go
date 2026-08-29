package unfaird

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

type UnfairdClient struct {
	BaseURL string
	HTTP    *http.Client
}

func New(baseURL string, httpClient *http.Client) *UnfairdClient {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &UnfairdClient{BaseURL: strings.TrimRight(baseURL, "/"), HTTP: httpClient}
}

type HealthResponse struct {
	Status string `json:"status"`
}

type DecryptResponse struct {
	Exit DecryptExit `json:"exit"`
}

type DecryptExit struct {
	Code          int32  `json:"code"`
	Stdout        string `json:"stdout"`
	Stderr        string `json:"stderr"`
	DownloadURL   string `json:"download_url"`
	ValidateUntil int64  `json:"validate_until"`
}

func (c *UnfairdClient) Health(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("health status %d: %s", resp.StatusCode, string(body))
	}
	var hr HealthResponse
	_ = json.NewDecoder(resp.Body).Decode(&hr)
	if hr.Status != "" && hr.Status != "ok" {
		return fmt.Errorf("unexpected health status: %s", hr.Status)
	}
	return nil
}

func (c *UnfairdClient) DecryptFile(ctx context.Context, ipaPath string) (DecryptResponse, error) {
	var out DecryptResponse
	f, err := os.Open(ipaPath)
	if err != nil {
		return out, err
	}
	defer f.Close()

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	part, err := w.CreateFormFile("ipa", filepath.Base(ipaPath))
	if err != nil {
		return out, err
	}
	if _, err := io.Copy(part, f); err != nil {
		return out, err
	}
	if err := w.Close(); err != nil {
		return out, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+"/api/v1/decrypt", &buf)
	if err != nil {
		return out, err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 300 {
		return out, fmt.Errorf("decrypt status %d: %s", resp.StatusCode, string(body))
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return out, fmt.Errorf("decode response: %w (%s)", err, string(body))
	}
	return out, nil
}

func (c *UnfairdClient) DownloadOutput(ctx context.Context, downloadURL, destPath string) error {
	url := downloadURL
	if strings.HasPrefix(url, "/") {
		url = c.BaseURL + url
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	resp, err := c.HTTP.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("download status %d: %s", resp.StatusCode, string(body))
	}
	if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
		return err
	}
	f, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

func JobIDFromDownloadURL(url string) string {
	// /api/v1/decrypt/<uuid>/output
	parts := strings.Split(strings.Trim(url, "/"), "/")
	for i := 0; i+1 < len(parts); i++ {
		if parts[i] == "decrypt" {
			return parts[i+1]
		}
	}
	return ""
}
