package install

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallMergesIntoExistingConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "claude_desktop_config.json")
	pre := `{
  "mcpServers": {
    "other": {"command": "elsewhere", "args": ["--flag"]}
  },
  "someOtherTopLevel": true
}`
	if err := os.WriteFile(path, []byte(pre), 0o644); err != nil {
		t.Fatal(err)
	}
	cc := ClientConfig{Path: path, mcpServersKey: "mcpServers"}
	entry := ServerEntry{Command: "/usr/local/bin/pickmem", Args: []string{"serve"}}
	if err := Install(cc, "pickmem", entry); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(got, &doc); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Other top-level fields must survive.
	if doc["someOtherTopLevel"] != true {
		t.Errorf("Install stomped on top-level fields: %v", doc)
	}
	servers, _ := doc["mcpServers"].(map[string]any)
	if _, ok := servers["other"]; !ok {
		t.Errorf("Install dropped 'other' server: %v", servers)
	}
	pickmem, ok := servers["pickmem"].(map[string]any)
	if !ok {
		t.Fatalf("pickmem entry missing: %v", servers)
	}
	if pickmem["command"] != "/usr/local/bin/pickmem" {
		t.Errorf("wrong command: %v", pickmem)
	}
}

func TestInstallCreatesFileWhenMissing(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "config.json")
	cc := ClientConfig{Path: path, mcpServersKey: "mcpServers"}
	if err := Install(cc, "pickmem", ServerEntry{Command: "pickmem", Args: []string{"serve"}}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("file not created: %v", err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if _, ok := doc["mcpServers"].(map[string]any)["pickmem"]; !ok {
		t.Errorf("pickmem entry missing in fresh config: %v", doc)
	}
}

func TestInstallReplacesExistingPickmemEntry(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	cc := ClientConfig{Path: path, mcpServersKey: "mcpServers"}
	if err := Install(cc, "pickmem", ServerEntry{Command: "old", Args: []string{"serve"}}); err != nil {
		t.Fatal(err)
	}
	if err := Install(cc, "pickmem", ServerEntry{Command: "new", Args: []string{"serve"}}); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	_ = json.Unmarshal(data, &doc)
	servers := doc["mcpServers"].(map[string]any)
	pickmem := servers["pickmem"].(map[string]any)
	if pickmem["command"] != "new" {
		t.Errorf("Install did not replace: got %v", pickmem)
	}
}

func TestUninstallRemovesEntryButKeepsOthers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cfg.json")
	cc := ClientConfig{Path: path, mcpServersKey: "mcpServers"}
	_ = Install(cc, "pickmem", ServerEntry{Command: "pickmem"})
	_ = Install(cc, "other", ServerEntry{Command: "other"})
	if err := Uninstall(cc, "pickmem"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	var doc map[string]any
	_ = json.Unmarshal(data, &doc)
	servers := doc["mcpServers"].(map[string]any)
	if _, ok := servers["pickmem"]; ok {
		t.Errorf("pickmem entry still present after Uninstall")
	}
	if _, ok := servers["other"]; !ok {
		t.Errorf("Uninstall removed the wrong entry")
	}
}

func TestUninstallOnMissingFileIsNoop(t *testing.T) {
	dir := t.TempDir()
	cc := ClientConfig{Path: filepath.Join(dir, "nope.json"), mcpServersKey: "mcpServers"}
	if err := Uninstall(cc, "pickmem"); err != nil {
		t.Errorf("Uninstall on missing file errored: %v", err)
	}
}

func TestResolveKnownClients(t *testing.T) {
	for _, name := range Clients() {
		cc, err := Resolve(name)
		if err != nil {
			// Cursor and Claude Desktop should both resolve on every
			// supported OS; if this test runs somewhere weird we still
			// don't want to hide it.
			t.Errorf("Resolve(%q): %v", name, err)
			continue
		}
		if cc.Path == "" {
			t.Errorf("Resolve(%q) returned empty Path", name)
		}
		if cc.DisplayName == "" {
			t.Errorf("Resolve(%q) returned empty DisplayName", name)
		}
	}
}

func TestResolveUnknownClientErrors(t *testing.T) {
	if _, err := Resolve("emacs"); err == nil {
		t.Error("Resolve of unknown client should error")
	}
}
