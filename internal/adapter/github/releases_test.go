package github

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLatestParsesRelease(t *testing.T) {
	var gotUA, gotAccept, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotUA, gotAccept, gotPath = r.Header.Get("User-Agent"), r.Header.Get("Accept"), r.URL.Path
		w.Write([]byte(`{"tag_name":"v1.0.3","name":"OPTIC 1.0.3","html_url":"https://github.com/andredeomondes/optic/releases/tag/v1.0.3","body":"notas","published_at":"2026-10-26T12:00:00Z"}`))
	}))
	defer srv.Close()

	c := Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "andredeomondes/optic", UserAgent: "OPTIC"}
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Tag != "v1.0.3" || rel.Name != "OPTIC 1.0.3" || rel.Notes != "notas" ||
		rel.URL != "https://github.com/andredeomondes/optic/releases/tag/v1.0.3" {
		t.Fatalf("unexpected release: %+v", rel)
	}
	if want := time.Date(2026, 10, 26, 12, 0, 0, 0, time.UTC); !rel.PublishedAt.Equal(want) {
		t.Fatalf("publishedAt = %v, want %v", rel.PublishedAt, want)
	}
	if gotUA != "OPTIC" || gotAccept != "application/vnd.github+json" {
		t.Fatalf("headers: UA=%q Accept=%q", gotUA, gotAccept)
	}
	if gotPath != "/repos/andredeomondes/optic/releases/latest" {
		t.Fatalf("path = %q", gotPath)
	}
}

func TestLatestErrorsOnRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "a/b", UserAgent: "OPTIC"}
	_, err := c.Latest(context.Background())
	if err == nil || !strings.Contains(err.Error(), "403") {
		t.Fatalf("err = %v, want one mentioning 403", err)
	}
}

func TestLatestErrorsWhenServerDown(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	url, hc := srv.URL, srv.Client()
	srv.Close()

	c := Client{HTTP: hc, BaseURL: url, Repo: "a/b", UserAgent: "OPTIC"}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("expected an error")
	}
}

func TestLatestDropsForeignReleaseURL(t *testing.T) {
	for _, bad := range []string{"file:///C:/x", "https://evil.com/", "https://github.com/other/repo/releases/1", "http://github.com/andredeomondes/optic/x"} {
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write([]byte(`{"tag_name":"v1.0.3","html_url":"` + bad + `"}`))
		}))
		c := Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "andredeomondes/optic", UserAgent: "OPTIC"}
		rel, err := c.Latest(context.Background())
		srv.Close()
		if err != nil {
			t.Fatal(err)
		}
		if rel.URL != "" {
			t.Fatalf("url %q kept as %q, want empty", bad, rel.URL)
		}
	}
}

func TestLatestRefusesCrossHostRedirect(t *testing.T) {
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"tag_name":"v9"}`))
	}))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer srv.Close()
	c := Client{BaseURL: srv.URL, Repo: "a/b", UserAgent: "OPTIC"}
	if _, err := c.Latest(context.Background()); err == nil {
		t.Fatal("expected redirect to be refused")
	}
}

func serveRelease(t *testing.T, body string) Client {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return Client{HTTP: srv.Client(), BaseURL: srv.URL, Repo: "andredeomondes/optic", UserAgent: "OPTIC"}
}

func TestLatestParsesAssetsAndDropsForeignURLs(t *testing.T) {
	c := serveRelease(t, `{"tag_name":"v1.1.0","assets":[
		{"name":"OPTIC-setup.exe","browser_download_url":"https://github.com/andredeomondes/optic/releases/download/v1.1.0/OPTIC-setup.exe","size":1234},
		{"name":"evil.exe","browser_download_url":"https://evil.example.com/andredeomondes/optic/releases/download/v1.1.0/evil.exe","size":1},
		{"name":"other.exe","browser_download_url":"https://github.com/someone/else/releases/download/v1/other.exe","size":2}]}`)
	rel, err := c.Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(rel.Assets) != 1 || rel.Assets[0].Name != "OPTIC-setup.exe" || rel.Assets[0].Size != 1234 ||
		rel.Assets[0].URL != "https://github.com/andredeomondes/optic/releases/download/v1.1.0/OPTIC-setup.exe" {
		t.Fatalf("assets = %+v", rel.Assets)
	}
}

func TestLatestWithoutAssetsGivesEmptySlice(t *testing.T) {
	rel, err := serveRelease(t, `{"tag_name":"v1.1.0"}`).Latest(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if rel.Assets == nil || len(rel.Assets) != 0 {
		t.Fatalf("assets = %#v, want empty non-nil slice", rel.Assets)
	}
}
