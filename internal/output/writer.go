package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
)

// Writer handles output to stdout and/or a file.
type Writer struct {
	mu       sync.Mutex
	file     *os.File
	jsonMode bool
	seen     map[string]bool
}

// NewWriter creates an output writer. If path is empty, only stdout is used.
func NewWriter(path string, jsonMode bool) (*Writer, error) {
	w := &Writer{
		jsonMode: jsonMode,
		seen:     make(map[string]bool),
	}
	if path != "" {
		f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
		if err != nil {
			return nil, fmt.Errorf("open output file: %w", err)
		}
		w.file = f
	}
	return w, nil
}

// WriteString writes a deduplicated line.
func (w *Writer) WriteString(line string) {
	line = strings.TrimSpace(line)
	if line == "" {
		return
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.seen[line] {
		return
	}
	w.seen[line] = true
	fmt.Println(line)
	if w.file != nil {
		_, _ = fmt.Fprintln(w.file, line)
	}
}

// WriteJSON marshals v and writes it as a JSON line.
func (w *Writer) WriteJSON(v any) {
	data, err := json.Marshal(v)
	if err != nil {
		return
	}
	w.WriteString(string(data))
}

// Close closes the output file if open.
func (w *Writer) Close() {
	if w.file != nil {
		_ = w.file.Close()
	}
}
