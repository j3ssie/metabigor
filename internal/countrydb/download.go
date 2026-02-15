package countrydb

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/j3ssie/metabigor/internal/output"
	"github.com/j3ssie/metabigor/public"
)

const (
	dbURL  = "https://github.com/iplocate/ip-address-databases/raw/refs/heads/main/ip-to-country/ip-to-country.csv.zip"
	dbDir  = ".metabigor"
	dbFile = "ip-to-country.csv"
)

// DBPath returns the path to the local country database.
func DBPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, dbDir, dbFile)
}

// EnsureLoaded returns a loaded DB, extracting from the embedded zip or downloading if needed.
func EnsureLoaded() (*DB, error) {
	path := DBPath()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		output.Info("Country database not found at %s, extracting from embedded data ...", path)
		if err := ExtractEmbedded(); err != nil {
			output.Warn("Embedded extraction failed: %v — downloading from GitHub ...", err)
			if err := Download(); err != nil {
				return nil, fmt.Errorf("auto-download country database: %w", err)
			}
		}
	}
	return Load(path)
}

// ExtractEmbedded extracts the CSV from the embedded zip to the local DB path.
func ExtractEmbedded() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, dbDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	data := public.CountryDB
	if len(data) == 0 {
		return fmt.Errorf("embedded country database is empty")
	}

	return unzipCSV(data, filepath.Join(dir, dbFile))
}

// Download fetches the latest country database from GitHub and stores it locally.
func Download() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}
	dir := filepath.Join(home, dbDir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create dir: %w", err)
	}

	output.Info("Downloading country database from GitHub (iplocate/ip-address-databases) ...")
	resp, err := http.Get(dbURL)
	if err != nil {
		return fmt.Errorf("download: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download returned status %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	return unzipCSV(data, filepath.Join(dir, dbFile))
}

// unzipCSV extracts the first .csv from a zip archive to dest.
func unzipCSV(zipData []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		return fmt.Errorf("open zip: %w", err)
	}

	for _, f := range zr.File {
		if filepath.Ext(f.Name) != ".csv" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open zip entry: %w", err)
		}
		defer func() { _ = rc.Close() }()

		out, err := os.Create(dest)
		if err != nil {
			return fmt.Errorf("create file: %w", err)
		}
		defer func() { _ = out.Close() }()

		n, err := io.Copy(out, rc)
		if err != nil {
			return fmt.Errorf("write file: %w", err)
		}
		output.Good("Country database saved to %s (%d MB)", dest, n/1024/1024)
		return nil
	}

	return fmt.Errorf("no CSV file found in zip archive")
}
