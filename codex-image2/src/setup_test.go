package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryConfigurationStore struct {
	settings  savedSettings
	key       string
	keyError  error
	writes    int
	failWrite int
}

func (s *memoryConfigurationStore) readSettings() (savedSettings, error) { return s.settings, nil }
func (s *memoryConfigurationStore) writeSettings(settings savedSettings) error {
	s.writes++
	if s.writes == s.failWrite {
		return errors.New("simulated settings write failure")
	}
	s.settings = settings
	return nil
}
func (s *memoryConfigurationStore) readKey() (string, error) {
	if s.key == "" {
		return "", errCredentialNotFound
	}
	return s.key, nil
}
func (s *memoryConfigurationStore) writeKey(key string) error {
	if s.keyError != nil {
		return s.keyError
	}
	s.key = key
	return nil
}

func emptyEnvironment(string) string { return "" }

func legacyEnvironment(name string) string {
	switch name {
	case "CODEX_API_URL":
		return "https://old.example.com/v1"
	case "CODEX_API_KEY":
		return "old-test-key"
	default:
		return ""
	}
}

func TestSecureConfigurationTakesPriorityAsPair(t *testing.T) {
	store := &memoryConfigurationStore{settings: savedSettings{APIURL: "https://new.example.com/v1"}, key: "new-test-key"}
	selected, err := selectConfiguration(store, legacyEnvironment)
	if err != nil {
		t.Fatal(err)
	}
	if selected.config.BaseURL != store.settings.APIURL || selected.config.APIKey != store.key || selected.source != "secure-settings" {
		t.Fatal("saved URL and key were not selected together")
	}
	status := configurationStatus(store, legacyEnvironment)
	if status["configured"] != true || status["initialized"] != false || status["network_checked"] != false {
		t.Fatal("legacy settings must be configured but unverified")
	}
	store.key = ""
	selected, err = selectConfiguration(store, legacyEnvironment)
	if err != nil || selected.config.APIKey != "" {
		t.Fatal("missing stored key must not fall back to the environment key")
	}
	if configurationStatus(store, legacyEnvironment)["configured"] != false {
		t.Fatal("missing key must not count as configured")
	}
}

func TestEnvironmentFallbackDoesNotUseStoredKey(t *testing.T) {
	store := &memoryConfigurationStore{key: "orphaned-stored-key"}
	status := configurationStatus(store, legacyEnvironment)
	if status["configured"] != true || status["initialized"] != false || status["configuration_source"] != "environment" {
		t.Fatal("complete environment configuration must remain unverified")
	}
	urlOnly := func(name string) string {
		if name == "CODEX_API_URL" {
			return "https://old.example.com/v1"
		}
		return ""
	}
	selected, err := selectConfiguration(store, urlOnly)
	if err != nil || selected.config.APIKey != "" {
		t.Fatal("must not mix environment URL with stored key")
	}
}

func TestInitializeOpensDialogAndVerifiesRealImage(t *testing.T) {
	imageBytes := testPNG(t)
	store := &memoryConfigurationStore{settings: savedSettings{APIURL: "https://old.example.com/v1", Initialized: true, VerifiedAt: "old", TestImage: "old.png"}, key: "old-test-key"}
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		if r.URL.Path != "/v1/images/generations" || r.Header.Get("Authorization") != "Bearer new-test-key" {
			t.Error("wrong endpoint or credentials")
		}
		if store.settings.Initialized {
			t.Error("initialized before the API returned an image")
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Error(err)
		}
		if payload["quality"] != "low" || payload["n"] != float64(1) {
			t.Error("setup must request one low-quality image")
		}
		fmt.Fprintf(w, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(imageBytes))
	}))
	defer server.Close()
	calls := 0
	dialog := func() (setupInput, error) { calls++; return setupInput{server.URL, "new-test-key"}, nil }
	out := filepath.Join(t.TempDir(), "test.png")
	result, err := initializeWithDialog(out, dialog, store)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || requests != 1 {
		t.Fatal("explicit initialization must open setup and actually generate once")
	}
	if result["initialized"] != true || !store.settings.Initialized || store.settings.VerifiedAt == "" || store.settings.TestImage != out {
		t.Fatal("successful image was not recorded")
	}
	raw, err := os.ReadFile(out)
	if err != nil || validateImage(raw) != nil {
		t.Fatal("test image must exist and decode")
	}
	selected, err := selectConfiguration(store, legacyEnvironment)
	if err != nil || selected.config.APIKey != "new-test-key" || selected.config.BaseURL != server.URL+"/v1" {
		t.Fatal("subsequent image generation would not use the verified pair")
	}
	status := configurationStatus(store, legacyEnvironment)
	if status["initialized"] != true || status["network_checked"] != false {
		t.Fatal("status must report historical verification, not a new network test")
	}
	serialized, _ := json.Marshal([]any{result, status, store.settings})
	if strings.Contains(string(serialized), "new-test-key") {
		t.Fatal("key leaked into output or settings")
	}
}

func TestFailedImageNeverCompletesInitialization(t *testing.T) {
	pngBytes := testPNG(t)
	tests := []struct {
		name   string
		status int
		body   string
	}{
		{"unauthorized", 401, `{"error":"do-not-echo-this-key"}`},
		{"forbidden", 403, `{}`},
		{"rate limited", 429, `{}`},
		{"server failure", 500, `{}`},
		{"empty data", 200, `{"data":[]}`},
		{"invalid json", 200, `<html>error</html>`},
		{"invalid base64", 200, `{"data":[{"b64_json":"%%%"}]}`},
		{"text disguised as image", 200, `{"data":[{"b64_json":"bm90LWFuLWltYWdl"}]}`},
		{"truncated PNG", 200, fmt.Sprintf(`{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(pngBytes[:len(pngBytes)-10]))},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			requests := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests++
				w.WriteHeader(test.status)
				fmt.Fprint(w, test.body)
			}))
			defer server.Close()
			store := &memoryConfigurationStore{settings: savedSettings{APIURL: "https://old.example.com/v1", Initialized: true, VerifiedAt: "old", TestImage: "old.png"}, key: "old-test-key"}
			out := filepath.Join(t.TempDir(), "result.png")
			result, err := initializeWithDialog(out, func() (setupInput, error) { return setupInput{server.URL, "new-test-key"}, nil }, store)
			if err == nil || result != nil || store.settings.Initialized {
				t.Fatal("failed test was treated as initialization success")
			}
			if requests != 1 {
				t.Fatal("setup must not automatically repeat billable requests")
			}
			if strings.Contains(err.Error(), "do-not-echo-this-key") {
				t.Fatal("server body leaked")
			}
			if configurationStatus(store, legacyEnvironment)["initialized"] != false {
				t.Fatal("failure left the previous success marker")
			}
			if _, err := os.Stat(out); !errors.Is(err, os.ErrNotExist) {
				t.Fatal("invalid image must not be saved")
			}
		})
	}
}

func TestSetupCancellationPreservesExistingConfiguration(t *testing.T) {
	before := savedSettings{APIURL: "https://old.example.com/v1", Initialized: true, VerifiedAt: "old", TestImage: "old.png"}
	store := &memoryConfigurationStore{settings: before, key: "old-test-key"}
	result, err := initializeWithDialog("", func() (setupInput, error) { return setupInput{}, errors.New("setup was cancelled") }, store)
	if err == nil || result != nil || store.settings != before || store.writes != 0 {
		t.Fatal("cancelled setup must fail without changing the existing configuration")
	}
}

func TestInterruptedSaveCannotUseMixedPair(t *testing.T) {
	for _, fail := range []string{"key", "settings"} {
		t.Run(fail, func(t *testing.T) {
			store := &memoryConfigurationStore{settings: savedSettings{APIURL: "https://old.example.com/v1", Initialized: true}, key: "old-test-key"}
			if fail == "key" {
				store.keyError = errors.New("simulated credential failure")
			} else {
				store.failWrite = 2
			}
			err := saveAPIConfig(apiConfig{"https://new.example.com/v1", "new-test-key"}, store)
			if err == nil || store.settings.Initialized || !store.settings.SetupPending {
				t.Fatal("save failure did not invalidate the configuration")
			}
			if _, err := selectConfiguration(store, legacyEnvironment); err == nil {
				t.Fatal("partial configuration must not be used or fall back to old environment values")
			}
		})
	}
}

func TestNoTestFlagCannotBypassInitialization(t *testing.T) {
	if err := runSetup([]string{"--no-test"}); err == nil {
		t.Fatal("setup --no-test must be rejected before opening the window")
	}
}

func TestVerificationMarkerWriteFailureIsNotSuccess(t *testing.T) {
	pngBytes := testPNG(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(pngBytes))
	}))
	defer server.Close()
	store := &memoryConfigurationStore{failWrite: 3}
	result, err := initializeWithDialog(filepath.Join(t.TempDir(), "test.png"), func() (setupInput, error) { return setupInput{server.URL, "new-test-key"}, nil }, store)
	if err == nil || result != nil || store.settings.Initialized {
		t.Fatal("initialization must fail if the verification marker cannot be saved")
	}
}

func TestSetupValidatesDownloadedImage(t *testing.T) {
	for _, valid := range []bool{true, false} {
		t.Run(fmt.Sprintf("valid=%v", valid), func(t *testing.T) {
			body := []byte("<html>not an image</html>")
			if valid {
				body = testPNG(t)
			}
			var server *httptest.Server
			server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/image.png" {
					if r.Header.Get("Authorization") != "" {
						t.Error("image download must not forward the API key")
					}
					_, _ = w.Write(body)
					return
				}
				fmt.Fprintf(w, `{"data":[{"url":%q}]}`, server.URL+"/image.png")
			}))
			defer server.Close()
			store := &memoryConfigurationStore{}
			result, err := initializeWithDialog(filepath.Join(t.TempDir(), "download.png"), func() (setupInput, error) { return setupInput{server.URL, "new-test-key"}, nil }, store)
			if valid && (err != nil || result["initialized"] != true) {
				t.Fatalf("valid URL image failed: %v", err)
			}
			if !valid && (err == nil || result != nil || store.settings.Initialized) {
				t.Fatal("invalid URL image passed initialization")
			}
		})
	}
}

func TestOutputWriteFailureIsNotInitializationSuccess(t *testing.T) {
	pngBytes := testPNG(t)
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		fmt.Fprintf(w, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(pngBytes))
	}))
	defer server.Close()
	parentFile := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(parentFile, []byte("existing user data"), 0600); err != nil {
		t.Fatal(err)
	}
	store := &memoryConfigurationStore{}
	result, err := initializeWithDialog(filepath.Join(parentFile, "test.png"), func() (setupInput, error) { return setupInput{server.URL, "new-test-key"}, nil }, store)
	if err == nil || result != nil || store.settings.Initialized || requests != 1 {
		t.Fatal("successful API response without a saved file is not initialization success")
	}
}

func TestLegacySettingsJSONRemainsUnverified(t *testing.T) {
	var settings savedSettings
	if err := json.Unmarshal([]byte(`{"api_url":"https://old.example.com/v1"}`), &settings); err != nil {
		t.Fatal(err)
	}
	store := &memoryConfigurationStore{settings: settings, key: "old-test-key"}
	if status := configurationStatus(store, emptyEnvironment); status["configured"] != true || status["initialized"] != false {
		t.Fatal("old settings cannot inherit a success marker")
	}
}
