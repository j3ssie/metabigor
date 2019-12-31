package core

// Options global options
type Options struct {
	Input      string
	Output     string
	TmpOutput  string
	ConfigFile string
	Proxy      string

	Concurrency int
	Delay       int
	SaveRaw     bool
	Timeout     int
	Verbose     bool
	Debug       bool
	// Ports       string
	// Rate        string
	Scan ScanOptions
	Net  NetOptions
}

// ScanOptions options for net command
type ScanOptions struct {
	Ports        string
	Rate         string
	Detail       bool
	Flat         bool
	SkipOverview bool
	TmpOutput    string
	NmapScripts  string
}

// NetOptions options for net command
type NetOptions struct {
	Asn string
	Org string
}

// HTTPRequest all information about response
type HTTPRequest struct {
	Method  string
	URL     string
	Headers map[string]string
	Body    string
}

// HTTPResponse all information about response
type HTTPResponse struct {
	StatusCode   int
	Status       string
	Headers      map[string][]string
	Body         string
	ResponseTime float64
	Length       int
	Beautify     string
}
