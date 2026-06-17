package version

import (
	"encoding/json"
	"testing"

	"golang.org/x/mod/semver"
)

func TestVersionParsedFromJSON(t *testing.T) {
	if Version == "dev" {
		t.Fatal("Version should be parsed from embedded JSON, got fallback \"dev\"")
	}
	if !semver.IsValid("v" + Version) {
		t.Fatalf("Version %q is not valid semver", Version)
	}
}

func TestPrereleaseConcatenation(t *testing.T) {
	info := versionInfo{Version: "1.0.0", Prerelease: "alpha"}
	data, _ := json.Marshal(info)

	var parsed versionInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	result := parsed.Version
	if parsed.Prerelease != "" {
		result += "-" + parsed.Prerelease
	}

	if result != "1.0.0-alpha" {
		t.Fatalf("expected \"1.0.0-alpha\", got %q", result)
	}
}

func TestEmptyPrerelease(t *testing.T) {
	info := versionInfo{Version: "2.0.0", Prerelease: ""}
	data, _ := json.Marshal(info)

	var parsed versionInfo
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}

	result := parsed.Version
	if parsed.Prerelease != "" {
		result += "-" + parsed.Prerelease
	}

	if result != "2.0.0" {
		t.Fatalf("expected \"2.0.0\", got %q", result)
	}
}
