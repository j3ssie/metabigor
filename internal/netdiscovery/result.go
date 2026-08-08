package netdiscovery

import "github.com/j3ssie/metabigor/internal/output"

// Result is one network range discovered for an input. It implements
// output.Record so a single lookup renders as text, flat, JSON, or CSV.
type Result struct {
	Input   string `json:"input"`
	ASN     string `json:"asn,omitempty"`
	CIDR    string `json:"cidr,omitempty"`
	Org     string `json:"org,omitempty"`
	Country string `json:"country,omitempty"`
	Source  string `json:"source,omitempty"`

	// Detail widens the text rendering to every populated column. It is a
	// presentation switch, so it never appears in JSON or CSV output.
	Detail bool `json:"-"`
}

// Value returns the primary result: the CIDR when known, otherwise the ASN.
func (r Result) Value() string {
	if r.CIDR != "" {
		return r.CIDR
	}
	return r.ASN
}

// Text renders one line, widened to every populated column under --detail.
func (r Result) Text() []string {
	if !r.Detail {
		return []string{r.Value()}
	}
	line := output.Columns(r.ASN, r.CIDR, r.Org, r.Country)
	if line == "" {
		return nil
	}
	return []string{line}
}

// Flat renders the bare CIDR (or ASN) for piping into other tools.
func (r Result) Flat() []string {
	if v := r.Value(); v != "" {
		return []string{v}
	}
	return nil
}

// CSV renders the full record regardless of --detail.
func (r Result) CSV() ([]string, [][]string) {
	return []string{"input", "asn", "cidr", "org", "country", "source"},
		[][]string{{r.Input, r.ASN, r.CIDR, r.Org, r.Country, r.Source}}
}
