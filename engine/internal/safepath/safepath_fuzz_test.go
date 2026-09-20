package safepath

import "testing"

func FuzzWithinResolved(f *testing.F) {
	f.Add("child/file.txt", "/tmp/parent")
	f.Add("..", "/tmp/parent")
	f.Add("", "")
	f.Fuzz(func(t *testing.T, candidate, parent string) {
		_ = WithinResolved(candidate, parent)
	})
}

func FuzzResolveExisting(f *testing.F) {
	f.Add("/tmp/example")
	f.Add("relative/path")
	f.Add("")
	f.Fuzz(func(t *testing.T, filename string) {
		_, _ = ResolveExisting(filename)
	})
}
