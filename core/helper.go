package core

import (
	"archive/zip"
	"bufio"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
)

// GetFileContent reads file and returns its content.
func GetFileContent(filename string) string {
	var result strings.Builder

	if strings.Contains(filename, "~") {
		var err error
		filename, err = homedir.Expand(filename)
		if err != nil {
			return ""
		}
	}

	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	// Create a buffer to store file content
	buf := make([]byte, 1024)

	// Read file content into the buffer
	for {
		n, err := file.Read(buf)
		if err != nil && err != io.EOF {
			return ""
		}
		if n == 0 {
			break
		}
		result.Write(buf[:n])
	}

	return result.String()
}

// ReadingFile Reading file and return content as []string
func ReadingFile(filename string) []string {
	var result []string
	if strings.HasPrefix(filename, "~") {
		filename, _ = homedir.Expand(filename)
	}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return result
	}

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := scanner.Text()
		result = append(result, val)
	}

	if err := scanner.Err(); err != nil {
		return result
	}
	return result
}

// ReadingFileUnique Reading file and return content as []string
func ReadingFileUnique(filename string) []string {
	var result []string
	if strings.Contains(filename, "~") {
		filename, _ = homedir.Expand(filename)
	}
	file, err := os.Open(filename)
	defer file.Close()
	if err != nil {
		return result
	}

	unique := true
	seen := make(map[string]bool)

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		val := scanner.Text()
		// unique stuff
		if val == "" {
			continue
		}
		val = strings.TrimSpace(val)
		if seen[val] && unique {
			continue
		}

		if unique {
			seen[val] = true
			result = append(result, val)
		}
	}

	if err := scanner.Err(); err != nil {
		return result
	}
	return result
}

// WriteToFile write string to a file
func WriteToFile(filename string, data string) (string, error) {
	file, err := os.OpenFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return "", err
	}
	defer file.Close()

	_, err = io.WriteString(file, data+"\n")
	if err != nil {
		return "", err
	}
	return filename, file.Sync()
}

// Unique unique content of a file and remove blank line
func Unique(filename string) {
	if filename == "" {
		return
	}
	DebugF("Unique Output: %v", filename)
	data := ReadingFileUnique(filename)
	WriteToFile(filename, strings.Join(data, "\n"))
}

// AppendToContent append string to a file
func AppendToContent(filename string, data string) (string, error) {
	// If the file doesn't exist, create it, or append to the file
	f, err := os.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return "", err
	}
	if _, err := f.Write([]byte(data + "\n")); err != nil {
		return "", err
	}
	if err := f.Close(); err != nil {
		return "", err
	}
	return filename, nil
}

// FileExists check if file is exist or not
func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

// FolderExists check if file is exist or not
func FolderExists(foldername string) bool {
	if _, err := os.Stat(foldername); os.IsNotExist(err) {
		return false
	}
	return true
}

// GetFileNames get all file name with extension
func GetFileNames(dir string, ext string) []string {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil
	}

	var files []string
	filepath.Walk(dir, func(path string, f os.FileInfo, _ error) error {
		if !f.IsDir() {
			if strings.HasSuffix(f.Name(), ext) {
				filename, _ := filepath.Abs(path)
				files = append(files, filename)
			}
		}
		return nil
	})
	return files
}

// IsJSON check if string is JSON or not
func IsJSON(str string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(str), &js) == nil
}

// GetTS get current timestamp and return a string
func GetTS() string {
	return strconv.FormatInt(time.Now().Unix(), 10)
}

// GenHash gen SHA1 hash from string
func GenHash(text string) string {
	h := sha1.New()
	h.Write([]byte(text))
	hashed := h.Sum(nil)
	return fmt.Sprintf("%x", hashed)
}

// Unzip will decompress a zip archive, moving all files and folders
// within the zip file (parameter 1) to an output directory (parameter 2).
func Unzip(src string, dest string) ([]string, error) {

	var filenames []string

	r, err := zip.OpenReader(src)
	if err != nil {
		return filenames, err
	}
	defer r.Close()

	for _, f := range r.File {

		// Store filename/path for returning and using later on
		fpath := filepath.Join(dest, f.Name)

		if !strings.HasPrefix(fpath, filepath.Clean(dest)+string(os.PathSeparator)) {
			return filenames, fmt.Errorf("%s: illegal file path", fpath)
		}

		filenames = append(filenames, fpath)

		if f.FileInfo().IsDir() {
			// Make Folder
			os.MkdirAll(fpath, os.ModePerm)
			continue
		}

		// Make File
		if err = os.MkdirAll(filepath.Dir(fpath), os.ModePerm); err != nil {
			return filenames, err
		}

		outFile, err := os.OpenFile(fpath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return filenames, err
		}

		rc, err := f.Open()
		if err != nil {
			return filenames, err
		}

		_, err = io.Copy(outFile, rc)

		// Close the file without defer to close before next iteration of loop
		outFile.Close()
		rc.Close()

		if err != nil {
			return filenames, err
		}
	}
	return filenames, nil
}

// ExpandLength make slice to length
func ExpandLength(list []string, length int) []string {
	c := []string{}
	for i := 1; i <= length; i++ {
		c = append(c, list[i%len(list)])
	}
	return c
}

// StartWithNum check if string start with number
func StartWithNum(raw string) bool {
	r, err := regexp.Compile("^[0-9].*")
	if err != nil {
		return false
	}
	return r.MatchString(raw)
}

// StripPath just strip some invalid string path
func StripPath(raw string) string {
	raw = strings.Replace(raw, "/", "_", -1)
	raw = strings.Replace(raw, " ", "_", -1)
	return raw
}

// Base64Encode just Base64 Encode
func Base64Encode(raw string) string {
	return base64.StdEncoding.EncodeToString([]byte(raw))
}

// Base64Decode just Base64 Encode
func Base64Decode(raw string) string {
	data, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return raw
	}
	return string(data)
}

// URLDecode decode url
func URLDecode(raw string) string {
	decodedValue, err := url.QueryUnescape(raw)
	if err != nil {
		return raw
	}
	return decodedValue
}

// URLEncode Encode query
func URLEncode(raw string) string {
	decodedValue := url.QueryEscape(raw)
	return decodedValue
}

// GenPorts gen list of ports based on input
func GenPorts(raw string) []string {
	var ports []string
	if strings.Contains(raw, ",") {
		items := strings.Split(raw, ",")
		for _, item := range items {
			if strings.Contains(item, "-") {
				min, err := strconv.Atoi(strings.Split(item, "-")[0])
				if err != nil {
					continue
				}
				max, err := strconv.Atoi(strings.Split(item, "-")[1])
				if err != nil {
					continue
				}
				for i := min; i <= max; i++ {
					ports = append(ports, fmt.Sprintf("%v", i))
				}
			} else {
				ports = append(ports, item)
			}
		}
	} else {
		if strings.Contains(raw, "-") {
			min, err := strconv.Atoi(strings.Split(raw, "-")[0])
			if err != nil {
				return ports
			}
			max, err := strconv.Atoi(strings.Split(raw, "-")[1])
			if err != nil {
				return ports
			}
			for i := min; i <= max; i++ {
				ports = append(ports, fmt.Sprintf("%v", i))
			}
		} else {
			ports = append(ports, raw)
		}
	}

	return ports
}
