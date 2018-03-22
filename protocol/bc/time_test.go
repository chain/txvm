package bc

import (
	"testing"
	"time"
)

func TestMillis(t *testing.T) {
	cases := []struct {
		t time.Time
		m uint64
	}{
		{t: time.Unix(0, 0).Add(time.Second).UTC(), m: 1000},
		{t: time.Unix(0, 0).Add(time.Minute).UTC(), m: 60000},
		{t: time.Unix(0, 0).Add(time.Hour).UTC(), m: 3600000},
		{t: time.Unix(0, 0).Add(time.Hour * 24).UTC(), m: 86400000},
	}

	for _, c := range cases {
		gotM := Millis(c.t)
		if gotM != c.m {
			t.Errorf("Millis(%v) = %d want %d", c.t, gotM, c.m)
		}

		gotT := FromMillis(c.m)
		if gotT != c.t {
			t.Errorf("FromMillis(%d) = %v want %v", c.m, gotT, c.t)
		}
	}
}

func TestDurationMillis(t *testing.T) {
	cases := []struct {
		d time.Duration
		m uint64
	}{
		{d: time.Second, m: 1000},
		{d: time.Minute, m: 60000},
		{d: time.Hour, m: 3600000},
		{d: time.Hour * 24, m: 86400000},
	}

	for _, c := range cases {
		gotM := DurationMillis(c.d)
		if gotM != c.m {
			t.Errorf("Millis(%v) = %d want %d", c.d, gotM, c.m)
		}

		gotD := MillisDuration(c.m)
		if gotD != c.d {
			t.Errorf("FromMillis(%d) = %v want %v", c.m, gotD, c.d)
		}
	}
}
