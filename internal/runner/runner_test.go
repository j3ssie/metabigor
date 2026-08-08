package runner

import (
	"os"
	"path/filepath"
	"slices"
	"testing"
)

func writeFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "targets.txt")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func TestReadInputsCollectsEverySource(t *testing.T) {
	path := writeFile(t, "from-file.com\n")

	got, err := ReadInputs("from-flag.com", path, []string{"from-arg.com"})
	if err != nil {
		t.Fatalf("ReadInputs: %v", err)
	}

	want := []string{"from-arg.com", "from-flag.com", "from-file.com"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestReadInputsSkipsBlanksCommentsAndDuplicates(t *testing.T) {
	path := writeFile(t, "a.com\n\n  \n# a comment\nb.com\na.com\n  c.com  \n")

	got, err := ReadInputs("", path, nil)
	if err != nil {
		t.Fatalf("ReadInputs: %v", err)
	}

	want := []string{"a.com", "b.com", "c.com"}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestReadInputsReportsMissingFile(t *testing.T) {
	_, err := ReadInputs("", filepath.Join(t.TempDir(), "nope.txt"), nil)
	if err == nil {
		t.Fatal("a missing --input-file should be an error, not a silent empty run")
	}
}

func TestReadInputsReturnsEmptyWhenNothingGiven(t *testing.T) {
	// Stdin is a terminal (or empty) under `go test`, so this must not block.
	got, err := ReadInputs("", "", nil)
	if err != nil {
		t.Fatalf("ReadInputs: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want no targets", got)
	}
}

func TestRunParallelVisitsEveryInput(t *testing.T) {
	inputs := []string{"a", "b", "c", "d", "e"}

	seen := make(chan string, len(inputs))
	RunParallel(inputs, 3, func(s string) { seen <- s })
	close(seen)

	var got []string
	for s := range seen {
		got = append(got, s)
	}
	slices.Sort(got)

	if !slices.Equal(got, inputs) {
		t.Errorf("got %v, want %v", got, inputs)
	}
}

func TestRunParallelToleratesZeroConcurrency(t *testing.T) {
	count := 0
	RunParallel([]string{"a", "b"}, 0, func(string) { count++ })

	if count != 2 {
		t.Errorf("got %d call(s), want 2", count)
	}
}
