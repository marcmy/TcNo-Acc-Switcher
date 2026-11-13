package app

import (
	"net/http"
	"os"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"TcNo-Acc-Switcher/internal/paths"
	"TcNo-Acc-Switcher/internal/platform"
)

func TestCompositeAssetHandlerSetsSecurityHeaders(t *testing.T) {
	handler := newCompositeAssetHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>test</title>")},
	})
	request := httptest.NewRequest(http.MethodGet, "http://wails.localhost/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	policy := response.Header().Get("Content-Security-Policy")
	for _, directive := range []string{
		"default-src 'none'",
		"object-src 'none'",
		"frame-ancestors 'none'",
		"script-src 'self'",
		"connect-src 'self'",
		"img-src 'self' data:",
		"media-src 'self'",
		"worker-src 'none'",
		"form-action 'none'",
		"sandbox allow-scripts allow-same-origin",
	} {
		if !strings.Contains(policy, directive) {
			t.Fatalf("CSP missing %q: %q", directive, policy)
		}
	}
	if got := response.Header().Get("Permissions-Policy"); !strings.Contains(got, "camera=()") || !strings.Contains(got, "clipboard-read=()") {
		t.Fatalf("Permissions-Policy = %q", got)
	}
	if got := response.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
	}
	if strings.Contains(policy, "https:") || strings.Contains(policy, "blob:") {
		t.Fatalf("CSP permits remote or blob content: %q", policy)
	}
	if got := response.Header().Get("Cross-Origin-Opener-Policy"); got != "same-origin" {
		t.Fatalf("Cross-Origin-Opener-Policy = %q", got)
	}
}

// Only /assets/ is content-hashed by Vite. img/ and backgrounds/ are the
// user-writable overlay and have to stay revalidated.
func TestImmutableCachingOnlyForHashedAssets(t *testing.T) {
	handler := newCompositeAssetHandler(fstest.MapFS{
		"index.html":         &fstest.MapFile{Data: []byte("<!doctype html>")},
		"assets/app-abc.js":  &fstest.MapFile{Data: []byte("export {}")},
		"img/platform/a.svg": &fstest.MapFile{Data: []byte("<svg/>")},
	})
	for path, want := range map[string]string{
		"/assets/app-abc.js":  "public, max-age=31536000, immutable",
		"/img/platform/a.svg": "",
		"/":                   "",
		"/assets/missing.js":  "public, max-age=31536000, immutable",
	} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "http://wails.localhost"+path, nil))
		if got := response.Header().Get("Cache-Control"); got != want {
			t.Errorf("Cache-Control for %s = %q, want %q", path, got, want)
		}
	}
}

func TestWritableAssetAllowedOnlyForMediaDirectories(t *testing.T) {
	for _, path := range []string{"img/profiles/avatar.webp", "backgrounds/app.png", "img/video.webm"} {
		if !writableAssetAllowed(path) {
			t.Errorf("writableAssetAllowed(%q) = false", path)
		}
	}
	for _, path := range []string{"index.html", "assets/app.js", "img/payload.js", "../img/avatar.png"} {
		if writableAssetAllowed(path) {
			t.Errorf("writableAssetAllowed(%q) = true", path)
		}
	}
}

// benchAssetHandler serves a small embedded set with the paths pointed at a
// temp dir, so the disk-override branch can resolve.
func benchAssetHandler(tb testing.TB) http.Handler {
	tb.Helper()
	exeDir := tb.TempDir()
	platform.ResetPathSingletonsForTest(exeDir)
	paths.ResetForTest(filepath.Join(exeDir, "TcNo Account Switcher"))
	return newCompositeAssetHandler(fstest.MapFS{
		"index.html":          &fstest.MapFile{Data: []byte("<!doctype html><title>t</title>")},
		"assets/index.js":     &fstest.MapFile{Data: []byte("export default 1;")},
		"img/placeholder.png": &fstest.MapFile{Data: []byte("png")},
	})
}

// BenchmarkAssetRequest covers both shapes the handler sees: a path that can
// never be served from disk, and an image that can.
func BenchmarkAssetRequest(b *testing.B) {
	handler := benchAssetHandler(b)

	for _, tc := range []struct{ name, path string }{
		{"Embedded", "/assets/index.js"},
		{"Image", "/img/placeholder.png"},
	} {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for b.Loop() {
				req := httptest.NewRequest(http.MethodGet, "http://wails.localhost"+tc.path, nil)
				handler.ServeHTTP(httptest.NewRecorder(), req)
			}
		})
	}
}

func TestCompositeAssetHandlerConfinesWritableMediaOverlay(t *testing.T) {
	exeDir := t.TempDir()
	platform.ResetPathSingletonsForTest(exeDir)
	paths.ResetForTest(filepath.Join(exeDir, "TcNo Account Switcher"))
	wwwroot, err := platform.WwwrootDir()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(wwwroot, "img"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wwwroot, "img", "normal.png"), []byte("normal-disk-image"), 0o644); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(exeDir, "private.png")
	if err := os.WriteFile(outside, []byte("private-outside-file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(wwwroot, "img", "escape.png")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	handler := newCompositeAssetHandler(fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<html></html>")},
		"img/escape.png": &fstest.MapFile{Data: []byte("embedded-safe-image")},
	})
	for _, tc := range []struct{ url, want string }{
		{"/img/normal.png", "normal-disk-image"},
		{"/img/escape.png", "embedded-safe-image"},
	} {
		rr := httptest.NewRecorder()
		handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "http://wails.localhost"+tc.url, nil))
		if rr.Code != http.StatusOK || !strings.Contains(rr.Body.String(), tc.want) {
			t.Errorf("%s: HTTP %d, body %q; want %q", tc.url, rr.Code, rr.Body.String(), tc.want)
		}
		if strings.Contains(rr.Body.String(), "private-outside-file") {
			t.Errorf("%s: exposed file outside wwwroot", tc.url)
		}
	}
}
