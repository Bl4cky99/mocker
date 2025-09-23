// SPDX-License-Identifier: MIT
// Copyright (c) 2025 Jason Giese (Bl4cky99)

package render

import (
	"bytes"
	"html/template"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Renderer struct {
	mu   sync.RWMutex
	tpls map[string]cachedTpl
}

type cachedTpl struct {
	tpl   *template.Template
	mtime time.Time
}

func New() *Renderer {
	return &Renderer{tpls: make(map[string]cachedTpl)}
}

func (r *Renderer) RenderString(tplSrc string, data any) ([]byte, error) {
	tpl, err := template.New("inline").Funcs(Funcs()).Option("missingkey=default").Parse(tplSrc)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func (r *Renderer) RenderFile(path string, data any) ([]byte, error) {
	abs, _ := filepath.Abs(path)
	info, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}

	r.mu.RLock()
	ct, ok := r.tpls[abs]
	r.mu.RUnlock()

	if !ok || ct.mtime.Before(info.ModTime()) {
		src, err := os.ReadFile(abs)
		if err != nil {
			return nil, err
		}

		tpl, err := template.New(filepath.Base(abs)).Funcs(Funcs()).Option("missingkey=default").Parse(string(src))
		if err != nil {
			return nil, err
		}

		ct = cachedTpl{tpl: tpl, mtime: info.ModTime()}

		r.mu.Lock()
		r.tpls[abs] = ct
		r.mu.Unlock()
	}

	var buf bytes.Buffer
	if err := ct.tpl.Execute(&buf, data); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
