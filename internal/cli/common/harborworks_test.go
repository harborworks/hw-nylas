package common

import (
	"encoding/json"
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
