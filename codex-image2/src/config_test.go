package main

import "testing"

func TestNormalizeAPIBase(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "root", input: "https://example.com", want: "https://example.com/v1"},
		{name: "existing v1", input: "https://example.com/v1/", want: "https://example.com/v1"},
		{name: "nested path", input: "https://example.com/openai", want: "https://example.com/openai/v1"},
		{name: "localhost http", input: "http://127.0.0.1:8080/v1", want: "http://127.0.0.1:8080/v1"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := normalizeAPIBase(test.input)
			if err != nil {
				t.Fatalf("normalizeAPIBase() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("normalizeAPIBase() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestNormalizeAPIBaseRejectsUnsafeValues(t *testing.T) {
	for _, input := range []string{
		"",
		"example.com",
		"http://example.com/v1",
		"https://user:password@example.com/v1",
		"https://example.com/v1?key=secret",
	} {
		if _, err := normalizeAPIBase(input); err == nil {
			t.Fatalf("normalizeAPIBase(%q) unexpectedly succeeded", input)
		}
	}
}

func TestEndpoint(t *testing.T) {
	got, err := endpoint("https://example.com/v1", "generations")
	if err != nil {
		t.Fatal(err)
	}
	if got != "https://example.com/v1/images/generations" {
		t.Fatalf("endpoint() = %q", got)
	}
}

func TestSelectSetupInitialURLHasNoProviderDefault(t *testing.T) {
	if got := selectSetupInitialURL("", ""); got != "" {
		t.Fatalf("selectSetupInitialURL() = %q, want an empty first-run value", got)
	}
	if got := selectSetupInitialURL(" https://environment.example/v1 ", "https://saved.example/v1"); got != "https://environment.example/v1" {
		t.Fatalf("environment URL did not take precedence: %q", got)
	}
	if got := selectSetupInitialURL("", " https://saved.example/v1 "); got != "https://saved.example/v1" {
		t.Fatalf("saved URL was not reused: %q", got)
	}
}

func TestHTTPStatusErrorDoesNotIncludeResponseBody(t *testing.T) {
	if got := httpStatusError(401); got != "API authentication failed; run setup again and check the API key" {
		t.Fatalf("unexpected status message: %q", got)
	}
}
