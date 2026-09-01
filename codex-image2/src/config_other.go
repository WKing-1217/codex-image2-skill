//go:build !windows

package main

import "errors"

func writeStoredAPIKey(string) error {
	return errors.New("the local secure setup wizard is currently available on Windows only; use CODEX_API_KEY on this platform")
}

func readStoredAPIKey() (string, error) {
	return "", errCredentialNotFound
}

func deleteStoredAPIKey() error {
	return errCredentialNotFound
}

func runSetupDialog() (setupInput, error) {
	return setupInput{}, errors.New("the local secure setup wizard is currently available on Windows only; configure CODEX_API_URL and CODEX_API_KEY locally")
}
