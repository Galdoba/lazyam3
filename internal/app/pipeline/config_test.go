package pipeline

import "testing"

// TestNewPRTTimestamp_Monotonic exercises the strictly-increasing guard by
// forcing every call to collide on the same clock second, then asserts each
// returned value is greater than the previous and stays a 12-digit string.
func TestNewPRTTimestamp_Monotonic(t *testing.T) {
	lastPRTSec = 0

	var prev string
	for i := 0; i < 20; i++ {
		got := newPRTTimestamp()
		if len(got) != 12 {
			t.Fatalf("expected 12-digit timestamp, got %q (len %d)", got, len(got))
		}
		if prev != "" && got <= prev {
			t.Fatalf("timestamp not strictly increasing: prev=%q got=%q", prev, got)
		}
		prev = got
	}
}
