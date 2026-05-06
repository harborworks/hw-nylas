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

	"github.com/nylas/cli/internal/domain"
)

type harborWorksCredentials struct {
	APIBaseURL string `json:"api_base_url"`
	Token      string `json:"token"`
}

type HarborWorksGrantStatus struct {
	ID        string          `json:"id"`
	Email     string          `json:"email"`
	Alias     string          `json:"alias,omitempty"`
	Provider  domain.Provider `json:"provider"`
	Status    string          `json:"status"`
	IsDefault bool            `json:"is_default"`
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
	result, ok, err := listHarborWorksGrants(identifier)
	if err != nil || !ok {
		return "", false, err
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

func ListHarborWorksGrants() ([]HarborWorksGrantStatus, bool, error) {
	result, ok, err := listHarborWorksGrants("")
	if err != nil || !ok {
		return nil, ok, err
	}

	grants := make([]HarborWorksGrantStatus, 0, len(result.Data))
	for _, grant := range result.Data {
		status := grant.Status
		if status == "active" {
			status = "valid"
		}
		grants = append(grants, HarborWorksGrantStatus{
			ID:        grant.GrantID,
			Email:     grant.Email,
			Alias:     grant.Alias,
			Provider:  domain.Provider(grant.Provider),
			Status:    status,
			IsDefault: grant.GrantID == result.DefaultGrantID,
		})
	}
	return grants, true, nil
}

type harborWorksGrantsResponse struct {
	DefaultGrantID string `json:"default_grant_id"`
	Data           []struct {
		GrantID  string `json:"grant_id"`
		Email    string `json:"email"`
		Alias    string `json:"alias"`
		Provider string `json:"provider"`
		Status   string `json:"status"`
	} `json:"data"`
}

func listHarborWorksGrants(identifier string) (*harborWorksGrantsResponse, bool, error) {
	creds, err := loadHarborWorksCredentials()
	if err != nil || creds == nil {
		return nil, false, err
	}

	endpoint, err := url.Parse(strings.TrimRight(creds.APIBaseURL, "/") + "/v1/nylas/grants")
	if err != nil {
		return nil, true, fmt.Errorf("parse Harbor Works API URL: %w", err)
	}
	if strings.TrimSpace(identifier) != "" {
		q := endpoint.Query()
		q.Set("identifier", identifier)
		endpoint.RawQuery = q.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, true, err
	}
	req.Header.Set("Authorization", "Bearer "+creds.Token)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, true, fmt.Errorf("Harbor Works grant lookup failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound && identifier != "" && !containsAt(identifier) {
		return nil, false, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, true, fmt.Errorf("Harbor Works grant lookup failed (%d): %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var result harborWorksGrantsResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, true, fmt.Errorf("decode Harbor Works grants: %w", err)
	}
	return &result, true, nil
}
