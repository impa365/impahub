package chatwoot

import "testing"

func TestNormalizeAttachmentURL(t *testing.T) {
	t.Run("keeps absolute URL", func(t *testing.T) {
		got := normalizeAttachmentURL("https://chatwoot.example.com", "https://cdn.example.com/file.jpg")
		want := "https://cdn.example.com/file.jpg"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("resolves relative path against base URL", func(t *testing.T) {
		got := normalizeAttachmentURL("https://chatwoot.example.com", "/rails/active_storage/blobs/abc/file.jpg")
		want := "https://chatwoot.example.com/rails/active_storage/blobs/abc/file.jpg"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})
}

func TestExtractRedirectHref(t *testing.T) {
	t.Run("extracts href from redirect HTML", func(t *testing.T) {
		html := `<html><body><a href="/rails/active_storage/blobs/redirected-file.jpg">Download</a></body></html>`
		got := extractRedirectHref(html)
		want := "/rails/active_storage/blobs/redirected-file.jpg"
		if got != want {
			t.Fatalf("expected %q, got %q", want, got)
		}
	})

	t.Run("returns empty string when href is missing", func(t *testing.T) {
		html := `<html><body>No link here</body></html>`
		if got := extractRedirectHref(html); got != "" {
			t.Fatalf("expected empty href, got %q", got)
		}
	})
}
