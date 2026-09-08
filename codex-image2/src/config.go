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
	APIURL       string `json:"api_url"`
	SetupPending bool   `json:"setup_pending,omitempty"`
	Initialized  bool   `json:"initialized"`
	VerifiedAt   string `json:"verified_at,omitempty"`
	TestImage    string `json:"test_image,omitempty"`
}

// Keep configuration access injectable so tests never touch a user's real key.
type configurationStore interface {
	readSettings() (savedSettings, error)
	writeSettings(savedSettings) error
	readKey() (string, error)
	writeKey(string) error
}

type localConfigurationStore struct{}

func (localConfigurationStore) readSettings() (savedSettings, error) { return readSettings() }
func (localConfigurationStore) writeSettings(s savedSettings) error  { return writeSettings(s) }
func (localConfigurationStore) readKey() (string, error)             { return readStoredAPIKey() }
func (localConfigurationStore) writeKey(key string) error            { return writeStoredAPIKey(key) }

type selectedConfiguration struct {
	config   apiConfig
	settings savedSettings
	source   string
}

func selectConfiguration(store configurationStore, getenv func(string) string) (selectedConfiguration, error) {
	settings, err := store.readSettings()
	if err != nil {
		return selectedConfiguration{}, err
	}
	if settings.SetupPending {
		return selectedConfiguration{}, errors.New("secure configuration save was interrupted; run setup again")
	}
	if strings.TrimSpace(settings.APIURL) != "" {
		key, err := store.readKey()
		if err != nil && !errors.Is(err, errCredentialNotFound) {
			return selectedConfiguration{}, err
		}
		// Never combine a saved URL with an environment key (or vice versa).
		return selectedConfiguration{apiConfig{settings.APIURL, strings.TrimSpace(key)}, settings, "secure-settings"}, nil
	}
	return selectedConfiguration{
		config: apiConfig{strings.TrimSpace(getenv("CODEX_API_URL")), strings.TrimSpace(getenv("CODEX_API_KEY"))},
		source: "environment",
	}, nil
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
	selected, err := selectConfiguration(localConfigurationStore{}, os.Getenv)
	if err != nil {
		return apiConfig{}, err
	}
	base, err := normalizeAPIBase(selected.config.BaseURL)
	if err != nil {
		return apiConfig{}, err
	}
	if requireKey && selected.config.APIKey == "" {
		return apiConfig{}, errors.New("API key is not configured; run setup and enter it in the local secure window")
	}
	return apiConfig{BaseURL: base, APIKey: selected.config.APIKey}, nil
}

func saveAPIConfig(config apiConfig, store configurationStore) error {
	base, err := normalizeAPIBase(config.BaseURL)
	if err != nil {
		return err
	}
	key := strings.TrimSpace(config.APIKey)
	if key == "" {
		return errors.New("API key must not be empty")
	}
	// Invalidate the old success marker BEFORE changing either member of the pair.
	// If a write fails, the pending state blocks accidental use of a mixed pair.
	if err := store.writeSettings(savedSettings{APIURL: base, SetupPending: true}); err != nil {
		return err
	}
	if err := store.writeKey(key); err != nil {
		return err
	}
	if err := store.writeSettings(savedSettings{APIURL: base}); err != nil {
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
	result := configurationStatus(localConfigurationStore{}, os.Getenv)
	result["cli_version"] = cliVersion
	path, err := settingsPath()
	if err == nil {
		result["settings_path"] = path
	}
	return result
}

func configurationStatus(store configurationStore, getenv func(string) string) map[string]any {
	result := map[string]any{"configured": false, "initialized": false, "network_checked": false}
	selected, err := selectConfiguration(store, getenv)
	if err != nil {
		result["configuration_error"] = err.Error()
		return result
	}
	result["configuration_source"] = selected.source
	base, err := normalizeAPIBase(selected.config.BaseURL)
	if err != nil {
		result["api_url_error"] = err.Error()
		return result
	}
	result["api_url"] = base
	if selected.source == "secure-settings" {
		result["api_url_source"] = "saved-settings"
		result["api_key_source"] = "windows-credential-manager"
	} else {
		result["api_url_source"] = "environment"
		result["api_key_source"] = "environment"
	}
	if selected.config.APIKey == "" {
		return result
	}
	result["configured"] = true
	settings := selected.settings
	if selected.source == "secure-settings" && settings.Initialized && settings.VerifiedAt != "" && settings.TestImage != "" {
		result["initialized"] = true
		result["verified_at"] = settings.VerifiedAt
		result["test_image"] = settings.TestImage
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
