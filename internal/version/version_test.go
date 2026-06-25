package version

import "testing"

// safeReleaseURL must honor only https (any host) or http to localhost, and ignore
// anything else (file://, link-local metadata IPs, arbitrary http hosts) so the
// KNIT_RELEASES_URL override can't be turned into an SSRF or local-file read.
func TestSafeReleaseURL(t *testing.T) {
	allowed := []string{
		"https://api.github.com/repos/rnwolfe/knit/releases/latest",
		"https://example.com/whatever",
		"http://localhost:8080/x",
		"http://127.0.0.1:0",
		"http://[::1]:9000/x",
	}
	for _, u := range allowed {
		if got := safeReleaseURL(u); got != u {
			t.Errorf("safeReleaseURL(%q) = %q, want allowed (unchanged)", u, got)
		}
	}

	disallowed := []string{
		"",
		"file:///etc/passwd",
		"http://169.254.169.254/latest/meta-data/",
		"http://example.com/releases", // plain http to a non-localhost host
		"ftp://example.com/x",
		"http://10.0.0.5/x",
		"gopher://localhost/x",
		"://nonsense",
	}
	for _, u := range disallowed {
		if got := safeReleaseURL(u); got != "" {
			t.Errorf("safeReleaseURL(%q) = %q, want \"\" (ignored)", u, got)
		}
	}
}
