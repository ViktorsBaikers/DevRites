package version

import "testing"

func TestCompare(t *testing.T) {
	cases := []struct {
		a, b   string
		cmp    int
		wantOK bool
	}{
		{"1.2.3", "1.2.3", 0, true},
		{"v1.2.4", "1.2.3", 1, true},
		{"1.2", "1.2.0", 0, true},
		{"1.10.0", "1.9.9", 1, true},
		{"2.0.0-rc.1", "2.0.0", 0, true},
		{"1.0.0+build", "1.0.1", -1, true},
		{"dev", "1.0.0", 0, false},
		{"1.x.0", "1.0.0", 0, false},
		{"", "1.0.0", 0, false},
	}
	for _, c := range cases {
		got, ok := Compare(c.a, c.b)
		if got != c.cmp || ok != c.wantOK {
			t.Errorf("Compare(%q, %q) = %d, %v; want %d, %v", c.a, c.b, got, ok, c.cmp, c.wantOK)
		}
	}
}
