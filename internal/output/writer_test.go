package output

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeRecord is a Record whose rendering is fully controlled by the test.
type fakeRecord struct {
	Name  string `json:"name"`
	Value string `json:"value"`

	text []string
	flat []string
}

func (r fakeRecord) Text() []string { return r.text }
func (r fakeRecord) Flat() []string { return r.flat }
func (r fakeRecord) CSV() ([]string, [][]string) {
	return []string{"name", "value"}, [][]string{{r.Name, r.Value}}
}

func newRecord(name, value string) fakeRecord {
	return fakeRecord{Name: name, Value: value, text: []string{name + " | " + value}, flat: []string{value}}
}

// capture builds a writer that renders into a buffer instead of stdout.
func capture(t *testing.T, format Format) (*Writer, *bytes.Buffer) {
	t.Helper()
	w, err := NewWriter("", format, false)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	buf := &bytes.Buffer{}
	w.stdout = buf
	return w, buf
}

func TestParseFormat(t *testing.T) {
	for _, name := range []string{"text", "json", "csv", "flat", "JSON", "  csv  "} {
		if _, err := ParseFormat(name); err != nil {
			t.Errorf("ParseFormat(%q) returned error: %v", name, err)
		}
	}

	_, err := ParseFormat("yaml")
	if err == nil {
		t.Fatal("ParseFormat(\"yaml\") should fail")
	}
	// The error has to tell the user what is valid, not just that they are wrong.
	for _, want := range []string{"text", "json", "csv", "flat"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q should list the %q format", err, want)
		}
	}
}

func TestWriteRendersEachFormat(t *testing.T) {
	tests := []struct {
		format Format
		want   string
	}{
		{FormatText, "cloudflare | 1.1.1.1\n"},
		{FormatFlat, "1.1.1.1\n"},
		{FormatJSON, `{"name":"cloudflare","value":"1.1.1.1"}` + "\n"},
		{FormatCSV, "name,value\ncloudflare,1.1.1.1\n"},
	}

	for _, tt := range tests {
		t.Run(string(tt.format), func(t *testing.T) {
			w, buf := capture(t, tt.format)
			w.Write(newRecord("cloudflare", "1.1.1.1"))

			if got := buf.String(); got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestWriteDeduplicates(t *testing.T) {
	w, buf := capture(t, FormatText)
	w.Write(newRecord("a", "1.1.1.1"))
	w.Write(newRecord("a", "1.1.1.1"))
	w.Write(newRecord("b", "8.8.8.8"))

	if lines := strings.Count(buf.String(), "\n"); lines != 2 {
		t.Errorf("got %d line(s), want 2:\n%s", lines, buf.String())
	}
	if w.Count() != 2 {
		t.Errorf("Count() = %d, want 2", w.Count())
	}
}

func TestCSVHeaderWrittenOnce(t *testing.T) {
	w, buf := capture(t, FormatCSV)
	w.Write(newRecord("a", "1.1.1.1"))
	w.Write(newRecord("b", "8.8.8.8"))

	if got := strings.Count(buf.String(), "name,value"); got != 1 {
		t.Errorf("header appeared %d time(s), want 1:\n%s", got, buf.String())
	}
}

func TestCSVQuotesSeparators(t *testing.T) {
	w, buf := capture(t, FormatCSV)
	w.Write(newRecord("Cloudflare, Inc.", `say "hi"`))

	want := "\"Cloudflare, Inc.\",\"say \"\"hi\"\"\""
	if !strings.Contains(buf.String(), want) {
		t.Errorf("got %q, want it to contain %q", buf.String(), want)
	}
}

func TestWriteSkipsEmptyRenderings(t *testing.T) {
	w, buf := capture(t, FormatFlat)
	w.Write(fakeRecord{flat: nil})
	w.Write(fakeRecord{flat: []string{"   "}})

	if buf.Len() != 0 {
		t.Errorf("expected no output, got %q", buf.String())
	}
}

func TestOutputFileTruncatesByDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")

	for range 2 {
		w, err := NewWriter(path, FormatFlat, false)
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		w.stdout = &bytes.Buffer{}
		w.Write(newRecord("a", "1.1.1.1"))
		if err := w.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := string(data); got != "1.1.1.1\n" {
		t.Errorf("second run should overwrite, got %q", got)
	}
}

func TestOutputFileAppends(t *testing.T) {
	path := filepath.Join(t.TempDir(), "out.txt")

	for range 2 {
		w, err := NewWriter(path, FormatFlat, true)
		if err != nil {
			t.Fatalf("NewWriter: %v", err)
		}
		w.stdout = &bytes.Buffer{}
		w.Write(newRecord("a", "1.1.1.1"))
		if err := w.Close(); err != nil {
			t.Fatalf("Close: %v", err)
		}
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := string(data); got != "1.1.1.1\n1.1.1.1\n" {
		t.Errorf("--append should keep both runs, got %q", got)
	}
}

// altRecord has a different CSV header from fakeRecord, to prove a run that
// emits two record types does not bury the second under the first's header.
type altRecord struct {
	Host string `json:"host"`
}

func (r altRecord) Text() []string { return []string{r.Host} }
func (r altRecord) Flat() []string { return []string{r.Host} }
func (r altRecord) CSV() ([]string, [][]string) {
	return []string{"host"}, [][]string{{r.Host}}
}

func TestCSVEmitsNewHeaderWhenColumnsChange(t *testing.T) {
	w, buf := capture(t, FormatCSV)
	w.Write(newRecord("a", "1.1.1.1"))
	w.Write(altRecord{Host: "example.com"})

	want := "name,value\na,1.1.1.1\nhost\nexample.com\n"
	if got := buf.String(); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestOutputFileUntouchedWhenNothingWritten(t *testing.T) {
	// A run that fails before producing results must not destroy the previous
	// results already in the file.
	path := filepath.Join(t.TempDir(), "out.txt")
	if err := os.WriteFile(path, []byte("previous run\n"), 0o600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	w, err := NewWriter(path, FormatFlat, false)
	if err != nil {
		t.Fatalf("NewWriter: %v", err)
	}
	w.stdout = &bytes.Buffer{}
	if err := w.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	if got := string(data); got != "previous run\n" {
		t.Errorf("empty run should leave the file alone, got %q", got)
	}
}
