package common

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type harborWorksCredentials struct {
	APIBaseURL string `json:"api_base_url"`
	Token      string `json:"token"`
}

func (c harborWorksCredentials) NylasBaseURL() string {
	return strings.TrimRight(c.APIBaseURL, "/") + "/v1/nylas"
}

func loadHarborWorksCredentials() (*harborWorksCredentials, error) {
	path, err := harborWorksCredentialsPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("read Harbor Works credentials: %w", err)
	}

	var creds harborWorksCredentials
	if err := json.Unmarshal(data, &creds); err != nil {
		return nil, fmt.Errorf("parse Harbor Works credentials: %w", err)
	}
	if strings.TrimSpace(creds.APIBaseURL) == "" || strings.TrimSpace(creds.Token) == "" {
		return nil, nil
	}
	return &creds, nil
}

func harborWorksCredentialsPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("determine home directory: %w", err)
	}
	profile := strings.TrimSpace(os.Getenv("HW_PROFILE"))
	if profile == "" || profile == "default" {
		return filepath.Join(home, ".hw", "credentials.json"), nil
	}
	return filepath.Join(home, ".hw", "profiles", profile+".json"), nil
}

func getHarborWorksGrantID(identifier string) (string, bool, error) {
	creds, err := loadHarborWorksCredentials()
	if err != nil || creds == nil {
		return "", false, err
	}

	endpoint, err := url.Parse(strings.TrimRight(creds.APIBaseURL, "/") + "/v1/nylas/grants")
	if err != nil {
		return "", true, fmt.Errorf("parse Harbor Works API URL: %w", err)
	}
	if strings.TrimSpace(identifier) != "" {
		q := endpoint.Query()
		q.Set("identifier", identifier)
		endpoint.RawQuery = q.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", true, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.Token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", true, fmt.Errorf("Harbor Works grant lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound && identifier != "" && !containsAt(identifier) {
		return "", false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return "", true, fmt.Errorf("Harbor Works grant lookup failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result struct {
		DefaultGrantID string `json:"default_grant_id"`
		Data           []struct {
			GrantID string `json:"grant_id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", true, fmt.Errorf("decode Harbor Works grants: %w", err)
	}
	if result.DefaultGrantID != "" {
		return result.DefaultGrantID, true, nil
	}
	if len(result.Data) > 0 && result.Data[0].GrantID != "" {
		return result.Data[0].GrantID, true, nil
	}

	return "", true, NewUserErrorWithSuggestions(
		"No Nylas account is connected in Harbor Works",
		"Connect an account from Harbor Works Integrations",
		"Then rerun this command with HW_PROFILE set to that profile",
	)
}
