// SPDX-License-Identifier: AGPL-3.0-only

package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"unicode/utf8"

	"github.com/gate4ai/sdk-go/abi"
)

// Settings — connectors.files в конфигурации.
type Settings struct {
	Roots    []string `json:"roots"`
	MaxBytes int64    `json:"max_bytes"`
}

func settings(m map[string]any) (Settings, error) {
	var s Settings
	b, _ := json.Marshal(m)
	if err := json.Unmarshal(b, &s); err != nil {
		return s, err
	}
	if len(s.Roots) == 0 {
		return s, errors.New("connectors.files.roots is empty: no folders are allowed")
	}
	if s.MaxBytes <= 0 {
		s.MaxBytes = 1 << 20
	}
	return s, nil
}

// Files — коннектор.
type Files struct{}

func sum(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}

func evidence(w abi.FSWriteResponse, data string) []abi.EvidenceItem {
	b, _ := json.Marshal(map[string]any{"path": w.RealPath, "size_before": w.SizeBefore, "size_after": w.SizeAfter, "data_sha256": sum([]byte(data))})
	return []abi.EvidenceItem{{Name: "file.json", MIME: "application/json", Data: b}}
}

// Execute исполняет действие.
func (Files) Execute(ctx context.Context, h *abi.Host, r abi.ExecRequest) (abi.ExecResult, error) {
	s, err := settings(r.Settings)
	if err != nil {
		return abi.ExecResult{}, err
	}
	p, _ := r.Args["path"].(string)
	switch r.Action {
	case "read":
		f, err := h.FSRead(p, s.MaxBytes)
		if err != nil {
			return abi.ExecResult{}, err
		}
		if !utf8.Valid(f.Data) || bytes.IndexByte(f.Data, 0) >= 0 {
			return abi.ExecResult{}, errors.New("binary files are not supported")
		}
		return abi.ExecResult{Result: map[string]any{"path": p, "content": string(f.Data), "size": f.Size, "truncated": f.Truncated}, ResultRef: f.RealPath}, nil
	case "list":
		l, err := h.FSList(p)
		if err != nil {
			return abi.ExecResult{}, err
		}
		out := []any{}
		for _, e := range l.Entries {
			item := map[string]any{"name": e.Name, "dir": e.Dir}
			if !e.Dir {
				item["size"] = e.Size
			}
			out = append(out, item)
		}
		return abi.ExecResult{Result: map[string]any{"path": p, "entries": out}, ResultRef: l.RealPath}, nil
	case "append":
		text, _ := r.Args["text"].(string)
		w, err := h.FSAppend(p, []byte(text))
		if err != nil {
			return abi.ExecResult{}, err
		}
		return abi.ExecResult{Result: map[string]any{"path": p, "appended_bytes": len(text)}, ResultRef: w.RealPath, Evidence: evidence(w, text)}, nil
	case "write":
		content, _ := r.Args["content"].(string)
		w, err := h.FSWrite(p, []byte(content))
		if err != nil {
			return abi.ExecResult{}, err
		}
		return abi.ExecResult{Result: map[string]any{"path": p, "bytes": len(content)}, ResultRef: w.RealPath, Evidence: evidence(w, content)}, nil
	}
	return abi.ExecResult{}, fmt.Errorf("unknown action %s", r.Action)
}

// Verify перечитывает файл: append — файл заканчивается добавленным текстом,
// write — содержимое совпадает побайтно.
func (Files) Verify(ctx context.Context, h *abi.Host, r abi.VerifyRequest) (abi.VerifyResult, error) {
	if _, err := settings(r.Settings); err != nil {
		return abi.VerifyResult{}, err
	}
	var want func([]byte) bool
	switch r.Action {
	case "append":
		text, _ := r.Args["text"].(string)
		want = func(b []byte) bool { return bytes.HasSuffix(b, []byte(text)) }
	case "write":
		content, _ := r.Args["content"].(string)
		want = func(b []byte) bool { return bytes.Equal(b, []byte(content)) }
	default:
		return abi.VerifyResult{Status: abi.Unverified, Method: "none"}, nil
	}
	p, _ := r.Args["path"].(string)
	f, err := h.FSRead(p, 64<<20)
	if err != nil {
		return abi.VerifyResult{Status: abi.Failed, Method: "read_back", Detail: err.Error()}, nil
	}
	if f.Truncated || !want(f.Data) {
		return abi.VerifyResult{Status: abi.Failed, Method: "read_back", Detail: "file content does not match"}, nil
	}
	return abi.VerifyResult{Status: abi.Verified, Method: "read_back", Detail: "sha256 " + sum(f.Data)}, nil
}
