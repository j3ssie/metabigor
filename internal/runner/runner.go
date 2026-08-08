// Package runner provides input handling and concurrent job execution utilities.
package runner

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/internal/output"
)

// ReadInputs gathers targets from positional arguments, --input, --input-file,
// and stdin, in that order. Blank lines, `#` comments, and duplicates are
// dropped. An --input-file of "-" reads stdin.
func ReadInputs(input, inputFile string, args []string) ([]string, error) {
	var (
		lines []string
		seen  = make(map[string]bool)
	)

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" || strings.HasPrefix(s, "#") || seen[s] {
			return
		}
		seen[s] = true
		lines = append(lines, s)
	}

	for _, a := range args {
		add(a)
	}
	if len(args) > 0 {
		output.Debug("Read %d target(s) from arguments", len(args))
	}

	if input != "" {
		add(input)
		output.Debug("Read target from --input: %s", input)
	}

	if inputFile != "" && inputFile != "-" {
		f, err := os.Open(inputFile)
		if err != nil {
			return nil, fmt.Errorf("read --input-file: %w", err)
		}
		count := scan(f, add)
		_ = f.Close()
		output.Debug("Read %d line(s) from %s", count, inputFile)
	}

	// Stdin is only consulted when no target was supplied any other way, or
	// when asked for explicitly with `-I -`. Reading it unconditionally would
	// block `metabigor net AS13335` forever whenever the shell hands the
	// process an open-but-idle stdin, which is common inside scripts and CI.
	if inputFile == "-" || (len(lines) == 0 && inputFile == "" && isPiped()) {
		count := scan(os.Stdin, add)
		output.Debug("Read %d line(s) from stdin", count)
	}

	output.Verbose("Collected %d unique target(s)", len(lines))
	return lines, nil
}

// scan feeds every line of r to add and returns the line count.
func scan(r io.Reader, add func(string)) int {
	count := 0
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		add(sc.Text())
		count++
	}
	return count
}

// isPiped reports whether stdin is a pipe or file rather than a terminal.
func isPiped() bool {
	info, err := os.Stdin.Stat()
	return err == nil && (info.Mode()&os.ModeCharDevice) == 0
}

// RunParallel runs fn for each input, with at most concurrency workers.
func RunParallel(inputs []string, concurrency int, fn func(string)) {
	if concurrency < 1 {
		concurrency = 1
	}
	output.Verbose("Starting %d worker(s) for %d target(s)", concurrency, len(inputs))

	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for _, in := range inputs {
		wg.Add(1)
		sem <- struct{}{}
		go func(val string) {
			defer wg.Done()
			defer func() { <-sem }()
			fn(val)
		}(in)
	}
	wg.Wait()
	output.Verbose("All workers finished")
}
