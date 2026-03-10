package app

import "testing"

func TestAddCountryEmojiPrefersLeftmostMatch(t *testing.T) {
	name := "nya-SG-ISIF[HK]-COP-BAGE-01[1.0x]"
	got := addCountryEmoji(name)
	want := "🇸🇬 " + name
	if got != want {
		t.Fatalf("addCountryEmoji() = %q, want %q", got, want)
	}
}

func TestWordIndexSkipsEmbeddedLetterMatches(t *testing.T) {
	idx, ok := wordIndex("bage-ru-just-ss-01", "us")
	if ok || idx != -1 {
		t.Fatalf("wordIndex() should ignore embedded letter matches, got idx=%d ok=%v", idx, ok)
	}
}
