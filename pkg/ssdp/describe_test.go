package ssdp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const sampleDesc = `<?xml version="1.0"?><root><device><friendlyName> [LG] webOS TV </friendlyName><modelName>OLED55C1</modelName></device></root>`

func TestParseDescription(t *testing.T) {
	name, model, err := ParseDescription([]byte(sampleDesc))
	if err != nil || name != "[LG] webOS TV" || model != "OLED55C1" {
		t.Errorf("got %q %q %v", name, model, err)
	}
	if _, _, err := ParseDescription([]byte("<root><device>")); err == nil {
		t.Error("want error on malformed xml")
	}
}

func TestDescribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/ok":
			_, _ = w.Write([]byte(sampleDesc))
		case "/redir":
			http.Redirect(w, r, "/ok", http.StatusFound)
		case "/big":
			_, _ = w.Write([]byte(strings.Repeat("a", maxDescriptionBytes+10)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()
	client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

	name, model, err := Describe(context.Background(), client, srv.URL+"/ok")
	if err != nil || name != "[LG] webOS TV" || model != "OLED55C1" {
		t.Errorf("ok: %q %q %v", name, model, err)
	}
	for _, p := range []string{"/redir", "/big", "/missing"} {
		if _, _, err := Describe(context.Background(), client, srv.URL+p); err == nil {
			t.Errorf("%s: want error", p)
		}
	}
}
