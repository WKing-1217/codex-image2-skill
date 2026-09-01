package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

const credentialTarget = "codex-image2/CODEX_API_KEY"

var errCredentialNotFound = errors.New("credential not found")

type apiConfig struct {
	BaseURL string
	APIKey  string
}

type savedSettings struct {
	APIURL string `json:"api_url"`
}

type setupInput struct {
	APIURL string `json:"api_url"`
	APIKey string `json:"api_key"`
}

func settingsPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", errors.New("could not find the user configuration directory")
	}
	return filepath.Join(dir, "CodexImage2", "config.json"), nil
}

func readSettings() (savedSettings, error) {
	path, err := settingsPath()
	if err != nil {
		return savedSettings{}, err
	}
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return savedSettings{}, nil
	}
	if err != nil {
		return savedSettings{}, errors.New("could not read the Codex Image2 settings")
	}
	var settings savedSettings
	if err := json.Unmarshal(raw, &settings); err != nil {
		return savedSettings{}, errors.New("the Codex Image2 settings file is invalid; run setup again")
	}
	return settings, nil
}

func writeSettings(settings savedSettings) error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return errors.New("could not create the Codex Image2 settings directory")
	}
	raw, _ := json.MarshalIndent(settings, "", "  ")
	raw = append(raw, '\n')
	if err := os.WriteFile(path, raw, 0600); err != nil {
		return errors.New("could not save the Codex Image2 settings")
	}
	return nil
}

func removeSettings() error {
	path, err := settingsPath()
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return errors.New("could not remove the Codex Image2 settings")
	}
	return nil
}

func normalizeAPIBase(value string) (string, error) {
	value = strings.TrimRight(strings.TrimSpace(value), "/")
	if value == "" {
		return "", errors.New("API URL is not configured; run setup first")
	}
	parsed, err := url.Parse(value)
	if err != nil || parsed.Host == "" || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		return "", errors.New("API URL must be a complete http:// or https:// URL")
	}
	if parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("API URL must not contain credentials, a query string, or a fragment")
	}
	if parsed.Scheme == "http" && parsed.Hostname() != "localhost" && parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "::1" {
		return "", errors.New("API URL must use HTTPS unless it points to localhost")
	}
	if !strings.HasSuffix(parsed.Path, "/v1") {
		parsed.Path = strings.TrimRight(parsed.Path, "/") + "/v1"
	}
	return strings.TrimRight(parsed.String(), "/"), nil
}

func endpoint(base, operation string) (string, error) {
	normalized, err := normalizeAPIBase(base)
	if err != nil {
		return "", err
	}
	return normalized + "/images/" + operation, nil
}

func resolveAPIConfig(requireKey bool) (apiConfig, error) {
	base := strings.TrimSpace(os.Getenv("CODEX_API_URL"))
	if base == "" {
		settings, err := readSettings()
		if err != nil {
			return apiConfig{}, err
		}
		base = settings.APIURL
	}
	base, err := normalizeAPIBase(base)
	if err != nil {
		return apiConfig{}, err
	}

	key := strings.TrimSpace(os.Getenv("CODEX_API_KEY"))
	if key == "" {
		key, err = readStoredAPIKey()
		if errors.Is(err, errCredentialNotFound) {
			err = nil
			key = ""
		}
		if err != nil {
			return apiConfig{}, err
		}
	}
	if requireKey && key == "" {
		return apiConfig{}, errors.New("API key is not configured; run setup and enter it in the local secure window")
	}
	return apiConfig{BaseURL: base, APIKey: key}, nil
}

func saveAPIConfig(config apiConfig) error {
	base, err := normalizeAPIBase(config.BaseURL)
	if err != nil {
		return err
	}
	key := strings.TrimSpace(config.APIKey)
	if key == "" {
		return errors.New("API key must not be empty")
	}
	if err := writeStoredAPIKey(key); err != nil {
		return err
	}
	if err := writeSettings(savedSettings{APIURL: base}); err != nil {
		return err
	}
	return nil
}

func clearAPIConfig() error {
	if err := deleteStoredAPIKey(); err != nil && !errors.Is(err, errCredentialNotFound) {
		return err
	}
	return removeSettings()
}

func configStatus() map[string]any {
	result := map[string]any{"configured": false}
	settings, settingsErr := readSettings()
	base := strings.TrimSpace(os.Getenv("CODEX_API_URL"))
	urlSource := "environment"
	if base == "" {
		base = settings.APIURL
		urlSource = "saved-settings"
	}
	if settingsErr != nil {
		result["settings_error"] = settingsErr.Error()
	}
	validBase := false
	if base != "" {
		normalized, err := normalizeAPIBase(base)
		if err != nil {
			result["api_url_error"] = err.Error()
		} else {
			base = normalized
			validBase = true
		}
		result["api_url"] = base
		result["api_url_source"] = urlSource
	}

	key := strings.TrimSpace(os.Getenv("CODEX_API_KEY"))
	keySource := "environment"
	if key == "" {
		var err error
		key, err = readStoredAPIKey()
		keySource = "windows-credential-manager"
		if err != nil && !errors.Is(err, errCredentialNotFound) {
			result["credential_error"] = err.Error()
		}
	}
	if key != "" && validBase {
		result["configured"] = true
		result["api_key_source"] = keySource
	}
	path, err := settingsPath()
	if err == nil {
		result["settings_path"] = path
	}
	return result
}

func validateSetupInput(input setupInput) (apiConfig, error) {
	base, err := normalizeAPIBase(input.APIURL)
	if err != nil {
		return apiConfig{}, err
	}
	key := strings.TrimSpace(input.APIKey)
	if key == "" {
		return apiConfig{}, errors.New("API key must not be empty")
	}
	return apiConfig{BaseURL: base, APIKey: key}, nil
}

func storedConfigurationSummary(config apiConfig) string {
	return fmt.Sprintf("API URL %s; API key stored as %s", config.BaseURL, credentialTarget)
}
