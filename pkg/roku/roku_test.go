package roku

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"
)

type recorder struct {
	mu    sync.Mutex
	calls []string
}

func (r *recorder) add(req *http.Request) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls = append(r.calls, req.Method+" "+req.URL.EscapedPath())
}

func newServer(t *testing.T, body string) (*Client, *recorder) {
	t.Helper()
	rec := &recorder{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.add(r)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return &Client{BaseURL: srv.URL}, rec
}

func TestKeypressPostsKey(t *testing.T) {
	c, rec := newServer(t, "")
	if err := c.Keypress(context.Background(), "Home"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"POST /keypress/Home"}; !reflect.DeepEqual(rec.calls, want) {
		t.Fatalf("calls = %v, want %v", rec.calls, want)
	}
}

func TestTextSendsEachRuneEscaped(t *testing.T) {
	c, rec := newServer(t, "")
	if err := c.Text(context.Background(), "ç &+"); err != nil {
		t.Fatal(err)
	}
	var want []string
	for _, r := range "ç &+" {
		want = append(want, "POST /keypress/Lit_"+strings.ReplaceAll(url.PathEscape(string(r)), "+", "%2B"))
	}
	if !reflect.DeepEqual(rec.calls, want) {
		t.Fatalf("calls = %v, want %v", rec.calls, want)
	}
	if rec.calls[0] != "POST /keypress/Lit_%C3%A7" || rec.calls[1] != "POST /keypress/Lit_%20" || rec.calls[3] != "POST /keypress/Lit_%2B" {
		t.Fatalf("unexpected escaping: %v", rec.calls)
	}
}

func TestTextStopsAtFirstError(t *testing.T) {
	var n int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n++
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	if err := (&Client{BaseURL: srv.URL}).Text(context.Background(), "abc"); err == nil {
		t.Fatal("expected error")
	}
	if n != 1 {
		t.Fatalf("requests = %d, want 1", n)
	}
}

func TestLaunch(t *testing.T) {
	c, rec := newServer(t, "")
	if err := c.Launch(context.Background(), "12"); err != nil {
		t.Fatal(err)
	}
	if want := []string{"POST /launch/12"}; !reflect.DeepEqual(rec.calls, want) {
		t.Fatalf("calls = %v, want %v", rec.calls, want)
	}
}

func TestAppsParses(t *testing.T) {
	c, rec := newServer(t, `<apps><app id="12" type="appl" version="5.1">Netflix</app><app id="837" type="appl" version="2.2">YouTube</app></apps>`)
	got, err := c.Apps(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := []App{{"12", "Netflix", "appl"}, {"837", "YouTube", "appl"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("apps = %v, want %v", got, want)
	}
	if rec.calls[0] != "GET /query/apps" {
		t.Fatalf("call = %v", rec.calls)
	}
}

func TestActiveAppHome(t *testing.T) {
	c, _ := newServer(t, `<active-app><app>Roku</app></active-app>`)
	got, err := c.ActiveApp(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if want := (App{Name: "Roku"}); got != want {
		t.Fatalf("app = %+v, want %+v", got, want)
	}

	c, _ = newServer(t, `<active-app><app id="12" type="appl">Netflix</app></active-app>`)
	got, err = c.ActiveApp(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "12" || got.Name != "Netflix" {
		t.Fatalf("app = %+v", got)
	}
}

const deviceXML = `<device-info>
<vendor-name>Roku</vendor-name>
<model-name>Roku Ultra</model-name>
<friendly-device-name>Sala</friendly-device-name>
<user-device-name>TV da Sala</user-device-name>
<software-version>12.5.0</software-version>
<power-mode>PowerOn</power-mode>
<wifi-mac>aa:aa:aa:aa:aa:aa</wifi-mac>
<ethernet-mac>bb:bb:bb:bb:bb:bb</ethernet-mac>
<is-tv>true</is-tv>
</device-info>`

func TestDeviceInfo(t *testing.T) {
	c, _ := newServer(t, deviceXML)
	d, err := c.DeviceInfo(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	want := DeviceInfo{
		Vendor: "Roku", Model: "Roku Ultra", FriendlyName: "Sala", UserName: "TV da Sala",
		Software: "12.5.0", PowerMode: "PowerOn", WifiMAC: "aa:aa:aa:aa:aa:aa",
		EthernetMAC: "bb:bb:bb:bb:bb:bb", IsTV: true,
	}
	if d != want {
		t.Fatalf("info = %+v, want %+v", d, want)
	}
	if d.Name() != "TV da Sala" || !d.On() || d.MAC() != "bb:bb:bb:bb:bb:bb" {
		t.Fatalf("helpers: %q %v %q", d.Name(), d.On(), d.MAC())
	}
	d.PowerMode = "DisplayOff"
	if d.On() {
		t.Fatal("DisplayOff must not be on")
	}
}

func TestDeviceInfoFallbacks(t *testing.T) {
	d := DeviceInfo{FriendlyName: "Sala", Model: "Ultra", WifiMAC: "w"}
	if d.Name() != "Sala" || d.MAC() != "w" {
		t.Fatalf("got %q %q", d.Name(), d.MAC())
	}
	d.FriendlyName = ""
	if d.Name() != "Roku Ultra" {
		t.Fatalf("name = %q", d.Name())
	}
}

func TestForbidden(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()
	err := (&Client{BaseURL: srv.URL}).Keypress(context.Background(), "Home")
	if !errors.Is(err, ErrForbidden) {
		t.Fatalf("err = %v, want ErrForbidden", err)
	}
}

func TestOtherStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	err := (&Client{BaseURL: srv.URL}).Launch(context.Background(), "1")
	if err == nil || err.Error() != "roku: HTTP 404" {
		t.Fatalf("err = %v", err)
	}
}

func TestTimeout(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	defer srv.Close()
	c := &Client{BaseURL: srv.URL, Timeout: 50 * time.Millisecond}
	start := time.Now()
	if err := c.Keypress(context.Background(), "Home"); err == nil {
		t.Fatal("expected timeout error")
	}
	if d := time.Since(start); d > 150*time.Millisecond {
		t.Fatalf("took %v", d)
	}
}

func TestBodyCapped(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("<apps>"))
		chunk := []byte(strings.Repeat(" ", 64*1024))
		for i := 0; i < 32; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
		_, _ = w.Write([]byte("</apps>"))
	}))
	defer srv.Close()
	if _, err := (&Client{BaseURL: srv.URL}).Apps(context.Background()); err == nil {
		t.Fatal("expected error for oversized body")
	}
}

func TestErrorsDoNotLeakPath(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
	}))
	timeoutErr := (&Client{BaseURL: srv.URL, Timeout: 30 * time.Millisecond}).Text(context.Background(), "x")
	srv.Close()
	closedErr := (&Client{BaseURL: srv.URL}).Text(context.Background(), "x")
	for _, err := range []error{timeoutErr, closedErr} {
		if err == nil {
			t.Fatal("expected error")
		}
		if s := err.Error(); strings.Contains(s, "Lit_") || strings.Contains(s, "/keypress") {
			t.Fatalf("error leaks path: %q", s)
		}
	}
}

func TestInvalidRequestErrorDoesNotLeakPath(t *testing.T) {
	err := (&Client{Host: "bad host"}).Text(context.Background(), "x")
	if err == nil {
		t.Fatal("expected error")
	}
	if s := err.Error(); strings.Contains(s, "Lit_") || strings.Contains(s, "/keypress") {
		t.Fatalf("error leaks path: %q", s)
	}
}

func TestRedirectNotFollowed(t *testing.T) {
	var hits int
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/x", http.StatusFound)
	}))
	defer srv.Close()
	err := (&Client{BaseURL: srv.URL}).Keypress(context.Background(), "Home")
	if err == nil || err.Error() != "roku: HTTP 302" {
		t.Fatalf("err = %v", err)
	}
	if hits != 0 {
		t.Fatalf("redirect target got %d requests", hits)
	}
}

func TestAppNameTrimmed(t *testing.T) {
	c, _ := newServer(t, "<active-app><app id=\"1\">\n  Netflix \n</app></active-app>")
	a, err := c.ActiveApp(context.Background())
	if err != nil || a.Name != "Netflix" {
		t.Fatalf("app = %+v, err = %v", a, err)
	}
}

func TestPostHasEmptyBody(t *testing.T) {
	var cl int64
	var body []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cl = r.ContentLength
		body, _ = io.ReadAll(r.Body)
	}))
	defer srv.Close()
	if err := (&Client{BaseURL: srv.URL}).Keypress(context.Background(), "Home"); err != nil {
		t.Fatal(err)
	}
	if cl != 0 || len(body) != 0 {
		t.Fatalf("content-length %d, body %q", cl, body)
	}
}
