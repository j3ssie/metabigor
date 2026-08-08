package output

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
	"sync"
)

// Writer serializes records to stdout and, optionally, to a file. Duplicate
// results are dropped so piping stays clean across concurrent workers.
type Writer struct {
	mu     sync.Mutex
	stdout io.Writer
	format Format
	seen   map[string]bool
	count  int

	file *os.File
	// truncate defers clearing the output file until the first result lands,
	// so a run that fails during setup leaves the previous results intact.
	truncate bool

	// csvHeader is the last header written, so a run that emits more than one
	// record type gets a fresh header instead of silently mismatched rows.
	csvHeader []string
	csvBuf    bytes.Buffer
	csvWriter *csv.Writer
}

// NewWriter creates a writer for the given format. An empty path writes to
// stdout only. An existing file is overwritten unless appendMode is set.
func NewWriter(path string, format Format, appendMode bool) (*Writer, error) {
	w := &Writer{
		stdout: os.Stdout,
		format: format,
		seen:   make(map[string]bool),
	}
	if path == "" {
		return w, nil
	}

	// Open without O_TRUNC and clear on first write instead: opening here
	// surfaces permission errors immediately, while deferring the truncate
	// keeps a failed run from destroying the previous results.
	flags := os.O_CREATE | os.O_WRONLY
	if appendMode {
		flags |= os.O_APPEND
	}
	f, err := os.OpenFile(path, flags, 0o644)
	if err != nil {
		return nil, fmt.Errorf("open output file %s: %w", path, err)
	}
	w.file = f
	w.truncate = !appendMode
	return w, nil
}

// Count returns how many unique results have been written.
func (w *Writer) Count() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.count
}

// Write renders rec in the writer's format.
func (w *Writer) Write(rec Record) {
	if rec == nil {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	switch w.format {
	case FormatJSON:
		data, err := json.Marshal(rec)
		if err != nil {
			Debug("cannot marshal record to JSON: %v", err)
			return
		}
		w.emit("", []string{string(data)})

	case FormatCSV:
		w.emitCSV(rec)

	case FormatFlat:
		w.emit("", rec.Flat())

	default:
		w.emit("", rec.Text())
	}
}

// emitCSV renders one record's rows, prefixed by a header row the first time
// that header is seen. Callers must hold w.mu.
func (w *Writer) emitCSV(rec Record) {
	header, rows := rec.CSV()
	if len(rows) == 0 {
		return
	}

	// Render the header only when it actually changes, rather than building it
	// once per record and discarding it.
	var headerLine string
	if len(header) > 0 && !slices.Equal(header, w.csvHeader) {
		w.csvHeader = slices.Clone(header)
		headerLine = w.csvLine(header)
	}

	lines := make([]string, 0, len(rows))
	for _, row := range rows {
		lines = append(lines, w.csvLine(row))
	}
	w.emit(headerLine, lines)
}

// emit writes lines as one deduplicated chunk, preceded by header when set.
// Callers must hold w.mu.
func (w *Writer) emit(header string, lines []string) {
	if len(lines) == 0 {
		return
	}
	chunk := strings.Join(lines, "\n")
	if strings.TrimSpace(chunk) == "" {
		return
	}
	if w.seen[chunk] {
		return
	}
	w.seen[chunk] = true
	w.count++

	if header != "" {
		w.print(header)
	}
	w.print(chunk)
}

// print writes s to stdout and the output file. Callers must hold w.mu.
func (w *Writer) print(s string) {
	_, _ = fmt.Fprintln(w.stdout, s)
	if w.file == nil {
		return
	}
	if w.truncate {
		w.truncate = false
		_ = w.file.Truncate(0)
		_, _ = w.file.Seek(0, io.SeekStart)
	}
	_, _ = fmt.Fprintln(w.file, s)
}

// csvLine renders one CSV row with correct quoting and escaping, reusing the
// writer's buffer so a million-row export does not allocate a csv.Writer per
// row. Callers must hold w.mu.
func (w *Writer) csvLine(fields []string) string {
	if w.csvWriter == nil {
		w.csvWriter = csv.NewWriter(&w.csvBuf)
	}
	w.csvBuf.Reset()
	_ = w.csvWriter.Write(fields)
	w.csvWriter.Flush()
	return strings.TrimRight(w.csvBuf.String(), "\r\n")
}

// Close releases the output file, if any. A run that wrote nothing leaves the
// file untouched, so a failed or empty run never destroys earlier results.
func (w *Writer) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.file == nil {
		return nil
	}
	err := w.file.Close()
	w.file = nil
	return err
}
