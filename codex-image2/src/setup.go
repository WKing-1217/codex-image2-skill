package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func initializeWithDialog(out string, dialog func() (setupInput, error), store configurationStore) (map[string]any, error) {
	// An explicit setup request ALWAYS opens the window, even for existing users.
	input, err := dialog()
	if err != nil {
		return nil, fmt.Errorf("initialization not completed: %w", err)
	}
	config, err := validateSetupInput(input)
	if err != nil {
		return nil, fmt.Errorf("initialization not completed: %w", err)
	}
	if err := saveAPIConfig(config, store); err != nil {
		return nil, fmt.Errorf("initialization not completed: %w", err)
	}
	if out == "" {
		out = filepath.Join(defaultOutDir, "setup-test-"+time.Now().Format("20060102-150405.000000000")+".png")
	}
	args := commonArgs{
		model: defaultModel, size: defaultSize, quality: "low", n: 1,
		outDir: defaultOutDir, maxAttempts: 1, timeout: 150 * time.Second,
	}
	// Use exactly the saved pair; environment variables cannot replace either value.
	data, err := generateConfigured("A friendly orange cat wearing a small astronaut helmet on the moon, cinematic light, no text", out, args, config)
	if err != nil {
		return nil, fmt.Errorf("initialization not completed; configuration saved but not verified; test image failed: %w", err)
	}
	outputs, ok := data["outputs"].([]string)
	if !ok || len(outputs) != 1 {
		return nil, errors.New("initialization not completed: missing test image output")
	}
	raw, err := os.ReadFile(outputs[0])
	if err != nil {
		return nil, errors.New("initialization not completed: could not read the saved test image")
	}
	if err := validateImage(raw); err != nil {
		return nil, fmt.Errorf("initialization not completed: saved test image failed validation: %w", err)
	}
	current, err := selectConfiguration(store, func(string) string { return "" })
	if err != nil || current.config != config {
		return nil, errors.New("initialization not completed: configuration changed during the test; run setup again")
	}
	verified := savedSettings{
		APIURL: config.BaseURL, Initialized: true,
		VerifiedAt: time.Now().UTC().Format(time.RFC3339), TestImage: outputs[0],
	}
	if err := store.writeSettings(verified); err != nil {
		return nil, fmt.Errorf("test image saved at %s, but initialization not completed: could not save verification status", outputs[0])
	}
	return map[string]any{
		"configured": true, "initialized": true, "network_checked": true,
		"api_url": config.BaseURL, "configuration_source": "secure-settings",
		"credential": credentialTarget, "summary": storedConfigurationSummary(config),
		"verified_at": verified.VerifiedAt, "test_image": data,
	}, nil
}
