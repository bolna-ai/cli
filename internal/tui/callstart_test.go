package tui

import "testing"

// e164Pattern gates the recipient number before a real outbound call is placed.
// A malformed number should be rejected at input, not turned into a wasted
// (billable) API round-trip. Pin the accept/reject boundary.
func TestE164Pattern(t *testing.T) {
	valid := []string{
		"+14155552671",
		"+911234567890",
		"+12",             // shortest allowed: + [1-9] then 1 digit
		"+123456789012345", // 15 digits, the E.164 max
	}
	for _, p := range valid {
		if !e164Pattern.MatchString(p) {
			t.Errorf("expected %q to be accepted", p)
		}
	}

	invalid := []string{
		"",
		"14155552671",       // missing +
		"+",                 // no digits
		"+0123456789",       // leading zero after +
		"+1",                // too short (needs at least 2 digits)
		"+1234567890123456", // 16 digits, over the E.164 max
		"+1 415 555 2671",   // spaces
		"+1-415-555-2671",   // dashes
		"+1(415)5552671",    // parens
		"garbage",
		"+abcdefghij",
	}
	for _, p := range invalid {
		if e164Pattern.MatchString(p) {
			t.Errorf("expected %q to be rejected", p)
		}
	}
}
