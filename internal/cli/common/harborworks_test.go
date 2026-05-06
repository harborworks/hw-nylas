package common

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestListHarborWorksGrants(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/nylas/grants" {
			t.Fatalf("path = %s, want /v1/nylas/grants", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-token" {
			t.Fatalf("Authorization = %q, want Bearer test-token", got)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"default_grant_id": "grant-1",
			"data": []map[string]any{
				{
					"grant_id": "grant-1",
					"email":    "ben@harborworks.ai",
					"alias":    "harborworks",
					"provider": "google",
					"status":   "active",
				},
			},
		})
	}))
	defer srv.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HW_PROFILE", "local")
	credentialsPath := filepath.Join(home, ".hw", "profiles", "local.json")
	if err := os.MkdirAll(filepath.Dir(credentialsPath), 0700); err != nil {
		t.Fatal(err)
	}
	credentials := map[string]string{
		"api_base_url": srv.URL,
		"token":        "test-token",
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	grants, ok, err := ListHarborWorksGrants()
	if err != nil {
		t.Fatalf("ListHarborWorksGrants error = %v", err)
	}
	if !ok {
		t.Fatal("ListHarborWorksGrants ok = false, want true")
	}
	if len(grants) != 1 {
		t.Fatalf("len(grants) = %d, want 1", len(grants))
	}
	grant := grants[0]
	if grant.ID != "grant-1" || grant.Email != "ben@harborworks.ai" || grant.Alias != "harborworks" || grant.Status != "valid" || !grant.IsDefault {
		t.Fatalf("unexpected grant: %+v", grant)
	}
}

func TestListHarborWorksGrantsMissingProfileReturnsProfileError(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Setenv("HW_PROFILE", "")

	_, ok, err := ListHarborWorksGrants()
	if err == nil {
		t.Fatal("ListHarborWorksGrants error = nil, want profile error")
	}
	if ok {
		t.Fatal("ListHarborWorksGrants ok = true, want false")
	}
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("error type = %T, want CLIError", err)
	}
	if cliErr.Message != `Harbor Works profile "default" is not configured for hw-nylas` {
		t.Fatalf("message = %q", cliErr.Message)
	}
}

func TestListHarborWorksGrantsHarborErrorStaysHarborWorksError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error":"not ready"}`))
	}))
	defer srv.Close()

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("HW_PROFILE", "default")
	credentialsPath := filepath.Join(home, ".hw", "credentials.json")
	if err := os.MkdirAll(filepath.Dir(credentialsPath), 0700); err != nil {
		t.Fatal(err)
	}
	credentials := map[string]string{
		"api_base_url": srv.URL,
		"token":        "test-token",
	}
	data, err := json.Marshal(credentials)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(credentialsPath, data, 0600); err != nil {
		t.Fatal(err)
	}

	_, ok, err := ListHarborWorksGrants()
	if err == nil {
		t.Fatal("ListHarborWorksGrants error = nil, want server error")
	}
	if !ok {
		t.Fatal("ListHarborWorksGrants ok = false, want true")
	}
	var cliErr *CLIError
	if !errors.As(err, &cliErr) {
		t.Fatalf("error type = %T, want CLIError", err)
	}
	if cliErr.Message != `Could not load Harbor Works Nylas accounts for profile "default"` {
		t.Fatalf("message = %q", cliErr.Message)
	}
	if cliErr.Code != ErrCodeAuthFailed {
		t.Fatalf("code = %q, want %q", cliErr.Code, ErrCodeAuthFailed)
	}
}
