package brand

import (
	"strings"
	"testing"
)

func TestBannerPlainHasIdentity(t *testing.T) {
	b := Banner("1.2.3", "abc1234", "2026-06-25", false)
	for _, want := range []string{Title, Tagline, "1.2.3", "abc1234", Author, Contact, URLProduct, URLRepo} {
		if !strings.Contains(b, want) {
			t.Errorf("plain banner missing %q\n%s", want, b)
		}
	}
	if strings.Contains(b, "\x1b[") {
		t.Error("plain banner (color=false) must not contain ANSI escapes")
	}
	if !strings.HasPrefix(b, "+--") {
		t.Errorf("plain banner should be an ASCII box, got:\n%s", b)
	}
}

func TestBannerStyledUsesRoundedBox(t *testing.T) {
	// The styled path renders a lipgloss rounded box (Unicode border). ANSI color
	// is gated on a real TTY by lipgloss, so it isn't asserted here.
	b := Banner("1.2.3", "abc1234", "2026-06-25", true)
	if !strings.Contains(b, "╭") || !strings.Contains(b, "╰") {
		t.Errorf("styled banner should use a rounded box:\n%s", b)
	}
	if !strings.Contains(b, "1.2.3") || !strings.Contains(b, URLRepo) {
		t.Errorf("styled banner missing version/url:\n%s", b)
	}
	if strings.HasPrefix(b, "+--") {
		t.Error("styled banner should not be the ASCII box")
	}
}
