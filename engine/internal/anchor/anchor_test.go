package anchor

import (
	"strings"
	"testing"
)

func TestSplitlines(t *testing.T) {
	got := Splitlines("a\r\nb\rc\n\nd e")
	want := []string{"a", "b", "c", "", "d", "e"}
	if strings.Join(got, "|") != strings.Join(want, "|") {
		t.Fatalf("Splitlines = %q", got)
	}
}

func TestAtAndFind(t *testing.T) {
	lines := Lines([]byte("def handler(request):\n    return request.user\nx = 1\n    return request.user\n"))
	cases := []struct {
		name       string
		start, end int
		quote      string
		want       Result
	}{
		{"exact", 2, 2, "return request.user", Found},
		{"whitespace only differs", 2, 2, "  return   request.user ", Found},
		{"spans lines", 1, 2, "(request): return request", Found},
		{"absent", 2, 2, "return request.admin", NotFound},
		{"empty", 2, 2, "   ", NotFound},
		{"too short", 2, 2, "return", TooShort},
		{"short quote is the whole window", 3, 3, "x = 1", Found},
	}
	for _, c := range cases {
		if got := At(lines, c.start, c.end, c.quote); got != c.want {
			t.Errorf("%s: At = %d, want %d", c.name, got, c.want)
		}
	}
	if got := Find(lines, "return request.user"); len(got) != 2 || got[0] != 2 || got[1] != 4 {
		t.Errorf("Find two sites = %v", got)
	}
	if got := Find(lines, "x = 1"); len(got) != 1 || got[0] != 3 {
		t.Errorf("Find short whole line = %v", got)
	}
	if got := Find(lines, "return"); len(got) != 0 {
		t.Errorf("Find too short = %v", got)
	}
}
