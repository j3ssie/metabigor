// Package runner provides input handling and concurrent job execution utilities.
package runner

import (
	"bufio"
	"os"
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/internal/output"
)

// ReadInputs gathers input from positional args, --input flag, -f file, and stdin.
func ReadInputs(input, inputFile string, args []string) []string {
	var lines []string
	seen := make(map[string]bool)

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			lines = append(lines, s)
		}
	}

	// Positional arguments (e.g. metabigor net Tesla)
	for _, a := range args {
		add(a)
	}
	if len(args) > 0 {
		output.Debug("Read %d input(s) from arguments", len(args))
	}

	// --input flag
	if input != "" {
		add(input)
		output.Debug("Read input from --input flag: %s", input)
	}

	// -f file
	if inputFile != "" {
		if f, err := os.Open(inputFile); err == nil {
			count := 0
			sc := bufio.NewScanner(f)
			for sc.Scan() {
				add(sc.Text())
				count++
			}
			_ = f.Close()
			output.Debug("Read %d lines from file: %s", count, inputFile)
		} else {
			output.Error("Failed to open input file %s: %v", inputFile, err)
		}
	}

	// Stdin (only if not a terminal)
	if info, err := os.Stdin.Stat(); err == nil && (info.Mode()&os.ModeCharDevice) == 0 {
		count := 0
		sc := bufio.NewScanner(os.Stdin)
		for sc.Scan() {
			add(sc.Text())
			count++
		}
		output.Debug("Read %d lines from stdin", count)
	}

	output.Verbose("Total inputs: %d (deduplicated)", len(lines))
	return lines
}

// RunParallel runs fn for each input with concurrency limit.
func RunParallel(inputs []string, concurrency int, fn func(string)) {
	if concurrency < 1 {
		concurrency = 1
	}
	output.Verbose("Starting %d worker(s) for %d input(s)", concurrency, len(inputs))

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
