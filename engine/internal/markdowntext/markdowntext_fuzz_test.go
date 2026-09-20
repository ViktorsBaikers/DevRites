package markdowntext

import (
	"bytes"
	"testing"
	"unicode/utf8"
)

func FuzzStructural(f *testing.F) {
	f.Add([]byte("# Title\n\n```go\nfmt.Println()\n```\n"))
	f.Add([]byte("~~~\nunclosed fence\n"))
	f.Add([]byte("   ```\n  ~~~\n"))
	f.Add([]byte("code ``` inline\n| not | a | fence |\n"))
	f.Add([]byte("line\r\nwith\r\ncrlf\r\n"))
	f.Add([]byte("\xff\xfe invalid utf8 \x00"))
	f.Add([]byte(""))
	f.Fuzz(func(t *testing.T, input []byte) {
		out, err := Structural(input)
		if err != nil {
			if utf8.Valid(input) && bytes.IndexByte(input, 0) < 0 {
				t.Fatalf("Structural rejected input without malformed UTF-8 or NUL: %v", err)
			}
			return
		}
		if len(out) != len(input) {
			t.Fatalf("Structural changed length: %d -> %d", len(input), len(out))
		}
		if !utf8.Valid(out) {
			t.Fatal("Structural emitted invalid UTF-8")
		}
		stable, err := Structural(out)
		if err != nil || !bytes.Equal(stable, out) {
			t.Fatal("Structural is not idempotent")
		}
	})
}
