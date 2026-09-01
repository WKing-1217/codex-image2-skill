package main

import (
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateConfiguredSavesImage(t *testing.T) {
	const key = "test-secret-never-log"
	wantImage := []byte("not-a-real-png-but-valid-api-bytes")
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		if request.URL.Path != "/v1/images/generations" {
			t.Errorf("request path = %q", request.URL.Path)
		}
		if got := request.Header.Get("Authorization"); got != "Bearer "+key {
			t.Errorf("Authorization header was not set correctly")
		}
		response.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(response, `{"data":[{"b64_json":%q}]}`, base64.StdEncoding.EncodeToString(wantImage))
	}))
	defer server.Close()

	out := filepath.Join(t.TempDir(), "result.png")
	args := commonArgs{
		model:       defaultModel,
		size:        defaultSize,
		quality:     defaultQuality,
		n:           1,
		outDir:      t.TempDir(),
		maxAttempts: 1,
		timeout:     5 * time.Second,
	}
	result, err := generateConfigured("test prompt", out, args, apiConfig{BaseURL: server.URL, APIKey: key})
	if err != nil {
		t.Fatal(err)
	}
	if result["model"] != defaultModel {
		t.Fatalf("result model = %v", result["model"])
	}
	gotImage, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotImage) != string(wantImage) {
		t.Fatalf("saved image bytes did not match API response")
	}
}

func TestGenerateConfiguredDoesNotExposeServerBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		response.WriteHeader(http.StatusForbidden)
		fmt.Fprint(response, `{"error":{"message":"server echoed test-secret-never-log"}}`)
	}))
	defer server.Close()

	args := commonArgs{
		model:       defaultModel,
		size:        defaultSize,
		quality:     defaultQuality,
		n:           1,
		outDir:      t.TempDir(),
		maxAttempts: 1,
		timeout:     5 * time.Second,
	}
	_, err := generateConfigured("test prompt", filepath.Join(t.TempDir(), "result.png"), args, apiConfig{BaseURL: server.URL, APIKey: "test-secret-never-log"})
	if err == nil {
		t.Fatal("generateConfigured() unexpectedly succeeded")
	}
	if got := err.Error(); got != "API access was denied; the account or subscription may not include gpt-image-2" {
		t.Fatalf("unexpected sanitized error: %q", got)
	}
}
