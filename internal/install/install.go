// Package install writes MCP server entries into the config files of
// the clients that support them (Claude Desktop, Cursor). All writes are
// merges — existing entries for other servers are preserved.
package install

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
)

// ClientConfig describes how to install the pickmem MCP server into one
// client's config file. Every supported client fits this shape because
// the MCP conventions have converged on `{"mcpServers":{"<name>":{...}}}`
// — differences are file location and whether the outer key varies.
type ClientConfig struct {
	// DisplayName is what the CLI prints (e.g. "Claude Desktop").
	DisplayName string
	// Path is the absolute config file path. Filled in by Resolve().
	Path string
	// mcpServersKey is the top-level object key we insert into (usually
	// "mcpServers").
	mcpServersKey string
}

// ServerEntry is the value that gets stored under mcpServers[name]. It's
// deliberately minimal — extra fields the client-specific schemas allow
// (env, cwd, disabled) get merged in later milestones if needed.
type ServerEntry struct {
	Command string            `json:"command"`
	Args    []string          `json:"args,omitempty"`
	Env     map[string]string `json:"env,omitempty"`
}

// Clients returns the sorted list of supported client names.
func Clients() []string {
	names := make([]string, 0, len(configs))
	for n := range configs {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// Resolve looks up a client by name and fills in the OS-specific config
// file path. Returns an error for unknown clients or unsupported OSes.
func Resolve(client string) (ClientConfig, error) {
	f, ok := configs[client]
	if !ok {
		return ClientConfig{}, fmt.Errorf("unknown client %q (supported: %v)", client, Clients())
	}
	cc, err := f()
	if err != nil {
		return ClientConfig{}, err
	}
	return cc, nil
}

// Install writes the pickmem server entry into the given client's config.
// name is the key under mcpServers (e.g. "pickmem"). Existing entries
// with the same name are replaced; other servers are untouched. Creates
// parent directories and the file itself if absent.
func Install(cc ClientConfig, name string, entry ServerEntry) error {
	raw, err := readOrEmpty(cc.Path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if len(raw) == 0 {
		doc = map[string]any{}
	} else {
		if err := json.Unmarshal(raw, &doc); err != nil {
			return fmt.Errorf("parse existing config %s: %w", cc.Path, err)
		}
	}
	servers, _ := doc[cc.mcpServersKey].(map[string]any)
	if servers == nil {
		servers = map[string]any{}
	}
	// Round-trip the entry through JSON so it lands as generic map[string]any
	// and merges cleanly with any hand-edited siblings.
	entryBytes, _ := json.Marshal(entry)
	var entryAny any
	_ = json.Unmarshal(entryBytes, &entryAny)
	servers[name] = entryAny
	doc[cc.mcpServersKey] = servers

	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	if err := os.MkdirAll(filepath.Dir(cc.Path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(cc.Path, out, 0o644)
}

// Uninstall removes the named entry from the client's config. No-op if
// the file or entry doesn't exist.
func Uninstall(cc ClientConfig, name string) error {
	raw, err := readOrEmpty(cc.Path)
	if err != nil || len(raw) == 0 {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return fmt.Errorf("parse existing config %s: %w", cc.Path, err)
	}
	servers, _ := doc[cc.mcpServersKey].(map[string]any)
	if servers == nil {
		return nil
	}
	if _, ok := servers[name]; !ok {
		return nil
	}
	delete(servers, name)
	doc[cc.mcpServersKey] = servers
	out, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')
	return os.WriteFile(cc.Path, out, 0o644)
}

// readOrEmpty reads the file, returning ([]byte{}, nil) if it doesn't
// exist. Any other error is surfaced.
func readOrEmpty(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	return data, err
}

// ---------- per-client config discovery ----------

// configs maps client names to functions that resolve their config path
// for the current OS. New clients get a new entry; the rest of the
// package doesn't care about client specifics.
var configs = map[string]func() (ClientConfig, error){
	"claude-desktop": claudeDesktop,
	"cursor":         cursor,
}

func claudeDesktop() (ClientConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ClientConfig{}, err
	}
	var path string
	switch runtime.GOOS {
	case "darwin":
		path = filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json")
	case "windows":
		// %APPDATA%\Claude\claude_desktop_config.json — os.UserConfigDir()
		// resolves that on Windows.
		cfg, err := os.UserConfigDir()
		if err != nil {
			return ClientConfig{}, err
		}
		path = filepath.Join(cfg, "Claude", "claude_desktop_config.json")
	case "linux":
		// Claude Desktop's linux install isn't official at the time of
		// writing; fall back to XDG so the flag isn't a dead-end when it
		// ships.
		cfg, err := os.UserConfigDir()
		if err != nil {
			return ClientConfig{}, err
		}
		path = filepath.Join(cfg, "Claude", "claude_desktop_config.json")
	default:
		return ClientConfig{}, fmt.Errorf("claude-desktop install not supported on %s", runtime.GOOS)
	}
	return ClientConfig{
		DisplayName:   "Claude Desktop",
		Path:          path,
		mcpServersKey: "mcpServers",
	}, nil
}

func cursor() (ClientConfig, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfig{
		DisplayName:   "Cursor",
		Path:          filepath.Join(home, ".cursor", "mcp.json"),
		mcpServersKey: "mcpServers",
	}, nil
}
