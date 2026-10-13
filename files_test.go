// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gate4ai/sdk-go/abi"
	"github.com/gate4ai/sdk-go/connector"
	"github.com/gate4ai/sdk-go/hosttest"
)

func setup(t *testing.T) (*hosttest.Host, connector.Connector, string) {
	home := t.TempDir()
	notes := filepath.Join(home, "Documents", "obsidian")
	os.MkdirAll(filepath.Join(notes, "sub"), 0o755)
	os.WriteFile(filepath.Join(notes, "a.md"), []byte("# A\n"), 0o644)
	os.WriteFile(filepath.Join(home, "secret.txt"), []byte("top secret"), 0o600)
	os.Symlink(filepath.Join(home, "secret.txt"), filepath.Join(notes, "link.md"))
	os.Symlink(home, filepath.Join(notes, "escape"))
	m, c, err := connector.Registered()
	if err != nil {
		t.Fatal(err)
	}
	h := hosttest.New(m, map[string]any{"roots": []any{"~/Documents/obsidian"}}, nil)
	h.Home = home
	return h, c, home
}

func TestReadListAppendWrite(t *testing.T) {
	h, c, home := setup(t)
	res, err := h.Execute(c, "read", map[string]any{"path": "~/Documents/obsidian/a.md"})
	if err != nil || res.Result.(map[string]any)["content"] != "# A\n" {
		t.Fatal(res, err)
	}
	res, err = h.Execute(c, "list", map[string]any{"path": "~/Documents/obsidian"})
	if err != nil || len(res.Result.(map[string]any)["entries"].([]any)) != 4 {
		t.Fatal(res, err)
	}
	args := map[string]any{"path": "~/Documents/obsidian/a.md", "text": "- item"}
	res, err = h.Execute(c, "append", args)
	if err != nil || len(res.Evidence) != 1 {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(filepath.Join(home, "Documents/obsidian/a.md"))
	if string(b) != "# A\n- item" {
		t.Fatal(string(b))
	}
	if v, _ := h.Verify(c, "append", args, res); v.Status != abi.Verified {
		t.Fatal(v)
	}
	// append создаёт новый файл в разрешённой папке.
	if _, err := h.Execute(c, "append", map[string]any{"path": "~/Documents/obsidian/sub/new.md", "text": "x"}); err != nil {
		t.Fatal(err)
	}
	wargs := map[string]any{"path": "~/Documents/obsidian/b.md", "content": "new file"}
	res, err = h.Execute(c, "write", wargs)
	if err != nil {
		t.Fatal(err)
	}
	if v, _ := h.Verify(c, "write", wargs, res); v.Status != abi.Verified {
		t.Fatal(v)
	}
	os.WriteFile(filepath.Join(home, "Documents/obsidian/b.md"), []byte("tampered"), 0o644)
	if v, _ := h.Verify(c, "write", wargs, res); v.Status != abi.Failed {
		t.Fatal("tamper not detected")
	}
}

func TestEscapes(t *testing.T) {
	h, c, home := setup(t)
	bad := []string{
		"~/secret.txt",
		"~/Documents/obsidian/../../secret.txt",
		"~/Documents/obsidian/link.md",
		"~/Documents/obsidian/escape/secret.txt",
		"Documents/obsidian/a.md",
		filepath.Join(home, "Documents", "obsidian2", "x"),
		"/etc/passwd",
	}
	for _, p := range bad {
		for _, action := range []string{"read", "append"} {
			args := map[string]any{"path": p}
			if action == "append" {
				args["text"] = "pwned"
			}
			_, err := h.Execute(c, action, args)
			if err == nil || !strings.Contains(err.Error(), abi.CodeCapabilityDenied) {
				t.Errorf("%s %s: %v", action, p, err)
			}
		}
	}
	if b, _ := os.ReadFile(filepath.Join(home, "secret.txt")); string(b) != "top secret" {
		t.Fatal("secret modified")
	}
	h.Settings = map[string]any{}
	if _, err := h.Execute(c, "read", map[string]any{"path": "~/x"}); err == nil || !strings.Contains(err.Error(), "roots") {
		t.Fatal("no roots must deny everything", err)
	}
	h.Settings = map[string]any{"roots": []any{"~/Documents/obsidian"}}
	os.WriteFile(filepath.Join(home, "Documents/obsidian/bin.dat"), []byte{0, 1, 2}, 0o644)
	if _, err := h.Execute(c, "read", map[string]any{"path": "~/Documents/obsidian/bin.dat"}); err == nil {
		t.Fatal("binary read")
	}
}

func TestManifest(t *testing.T) {
	m, _, _ := connector.Registered()
	if w, _ := m.Find("write"); w.Class != "write" {
		t.Fatal("write must default to ask")
	}
	if m.Runtime != "wasm" || len(m.Capabilities.Files) != 1 || m.Capabilities.Network != nil {
		t.Fatal("files connector must have only file capabilities", m.Capabilities)
	}
}
