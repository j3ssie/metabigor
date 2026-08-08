package urlsource

import "github.com/j3ssie/metabigor/internal/output"

// Result is one discovered URL, tagged with the source that surfaced it and
// whatever metadata that source happened to carry.
//
// Status, MIME, and Timestamp are best-effort: the CDX indexes provide all
// three, urlscan provides status and MIME, and OTX provides only a status.
// Anything a source does not report stays empty rather than being guessed.
type Result struct {
	Input     string `json:"input"`
	URL       string `json:"url"`
	Source    Source `json:"source"`
	Status    string `json:"status,omitempty"`
	MIME      string `json:"mime,omitempty"`
	Timestamp string `json:"timestamp,omitempty"`

	// Detail mirrors --detail so Text can widen without a second type.
	Detail bool `json:"-"`
}

// Text renders the bare URL, or a tagged line under --detail. Fields the source
// did not report are left out rather than rendered as empty columns.
func (r Result) Text() []string {
	if !r.Detail {
		return []string{r.URL}
	}
	return []string{output.Columns(r.URL, string(r.Source), r.Status, r.MIME, r.Timestamp)}
}

// Flat renders the bare URL, ignoring --detail so pipelines stay stable.
func (r Result) Flat() []string {
	return []string{r.URL}
}

// CSV renders the URL with its source and metadata columns.
func (r Result) CSV() ([]string, [][]string) {
	return []string{"input", "url", "source", "status", "mime", "timestamp"},
		[][]string{{r.Input, r.URL, string(r.Source), r.Status, r.MIME, r.Timestamp}}
}
