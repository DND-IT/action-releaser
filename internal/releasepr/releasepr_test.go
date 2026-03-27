package releasepr

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectMerge_NotPREvent(t *testing.T) {
	t.Setenv("GITHUB_EVENT_NAME", "push")
	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil manifest for non-PR event")
	}
}

func TestDetectMerge_PRNotMerged(t *testing.T) {
	event := map[string]interface{}{
		"action": "closed",
		"pull_request": map[string]interface{}{
			"merged": false,
			"head":   map[string]string{"ref": "release/v1.0.0"},
			"labels": []map[string]string{{"name": LabelPending}},
		},
	}
	eventFile := writeEventFile(t, event)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil manifest for non-merged PR")
	}
}

func TestDetectMerge_NoLabel(t *testing.T) {
	event := map[string]interface{}{
		"action": "closed",
		"pull_request": map[string]interface{}{
			"merged": true,
			"head":   map[string]string{"ref": "release/v1.0.0"},
			"labels": []map[string]string{{"name": "some-other-label"}},
		},
	}
	eventFile := writeEventFile(t, event)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil manifest for PR without pending label")
	}
}

func TestDetectMerge_WrongBranch(t *testing.T) {
	event := map[string]interface{}{
		"action": "closed",
		"pull_request": map[string]interface{}{
			"merged": true,
			"head":   map[string]string{"ref": "feature/something"},
			"labels": []map[string]string{{"name": LabelPending}},
		},
	}
	eventFile := writeEventFile(t, event)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m != nil {
		t.Fatal("expected nil manifest for non-release branch")
	}
}

func TestDetectMerge_ValidMerge(t *testing.T) {
	event := map[string]interface{}{
		"action": "closed",
		"pull_request": map[string]interface{}{
			"merged": true,
			"head":   map[string]string{"ref": "release/v1.2.0"},
			"labels": []map[string]string{{"name": LabelPending}},
		},
	}
	eventFile := writeEventFile(t, event)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected manifest for valid release PR merge")
	}
	if m.Tag != "v1.2.0" {
		t.Errorf("tag = %q, want v1.2.0", m.Tag)
	}
}

func TestDetectMerge_WithManifestFile(t *testing.T) {
	event := map[string]interface{}{
		"action": "closed",
		"pull_request": map[string]interface{}{
			"merged": true,
			"head":   map[string]string{"ref": "release/v2.0.0"},
			"labels": []map[string]string{{"name": LabelPending}},
		},
	}
	eventFile := writeEventFile(t, event)
	t.Setenv("GITHUB_EVENT_NAME", "pull_request")
	t.Setenv("GITHUB_EVENT_PATH", eventFile)

	// Write a manifest file in the working directory.
	dir := t.TempDir()
	orig, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(orig)

	manifest := Manifest{Version: "2.0.0", Strategy: "semver", Tag: "v2.0.0"}
	data, _ := json.Marshal(manifest)
	os.WriteFile(ManifestFile, data, 0644)

	m, err := DetectMerge()
	if err != nil {
		t.Fatal(err)
	}
	if m == nil {
		t.Fatal("expected manifest")
	}
	if m.Version != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0", m.Version)
	}
	if m.Strategy != "semver" {
		t.Errorf("strategy = %q, want semver", m.Strategy)
	}
}

func TestFormatPRBody(t *testing.T) {
	body := formatPRBody("1.2.0", "### Features\n- Added foo")
	if !strings.Contains(body, "1.2.0") {
		t.Error("body should contain version")
	}
	if !strings.Contains(body, "### Features") {
		t.Error("body should contain changelog")
	}
	if !strings.Contains(body, "action-releaser") {
		t.Error("body should mention action-releaser")
	}
}

func TestFormatPRBody_EmptyChangelog(t *testing.T) {
	body := formatPRBody("1.0.0", "")
	if !strings.Contains(body, "No changelog entries") {
		t.Error("body should indicate empty changelog")
	}
}

func TestManifestRoundtrip(t *testing.T) {
	m := Manifest{Version: "1.0.0", Strategy: "semver", Tag: "v1.0.0"}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}

	var m2 Manifest
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatal(err)
	}
	if m2.Version != m.Version || m2.Strategy != m.Strategy || m2.Tag != m.Tag {
		t.Errorf("roundtrip failed: got %+v, want %+v", m2, m)
	}
}

func writeEventFile(t *testing.T, event interface{}) string {
	t.Helper()
	data, err := json.Marshal(event)
	if err != nil {
		t.Fatal(err)
	}
	f := filepath.Join(t.TempDir(), "event.json")
	if err := os.WriteFile(f, data, 0644); err != nil {
		t.Fatal(err)
	}
	return f
}
