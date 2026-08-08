package netdiscovery

import "testing"

func TestValidBGPRowsDropsNonNetworkRows(t *testing.T) {
	rows := []BGPHEResult{
		{Result: "AS13335"},
		{Result: "1.1.1.0/24"},
		{Result: "2606:4700::/32"},
		{Result: "1"},                              // a row counter
		{Result: "104.16.132.229, 104.16.133.229"}, // DNS answers
		{Result: ""},
	}

	got := validBGPRows(rows)
	if len(got) != 3 {
		t.Fatalf("kept %d row(s), want 3: %+v", len(got), got)
	}
	for i, want := range []string{"AS13335", "1.1.1.0/24", "2606:4700::/32"} {
		if got[i].Result != want {
			t.Errorf("row %d = %q, want %q", i, got[i].Result, want)
		}
	}
}

func TestBGPResultsClassifiesASNsAndRoutes(t *testing.T) {
	got := bgpResults([]BGPHEResult{
		{Result: "as13335", Description: "Cloudflare", Country: "United States"},
		{Result: "1.1.1.0/24", Description: "Cloudflare", Country: "Unknown"},
		{Result: "not-a-network"},
	})

	if len(got) != 2 {
		t.Fatalf("got %d result(s), want 2: %+v", len(got), got)
	}

	if got[0].ASN != "AS13335" {
		t.Errorf("ASN = %q, want it normalized to AS13335", got[0].ASN)
	}
	if got[0].CIDR != "" {
		t.Errorf("an ASN row should not set CIDR, got %q", got[0].CIDR)
	}
	if got[0].Country != "United States" {
		t.Errorf("Country = %q", got[0].Country)
	}

	if got[1].CIDR != "1.1.1.0/24" {
		t.Errorf("CIDR = %q", got[1].CIDR)
	}
	// "Unknown" is bgp.he.net's placeholder, not a country.
	if got[1].Country != "" {
		t.Errorf("Country = %q, want it dropped", got[1].Country)
	}
}

func TestCIDRResultsSkipsBlanks(t *testing.T) {
	got := cidrResults([]string{"1.1.1.0/24", "  ", "", " 8.8.8.0/24 "})
	if len(got) != 2 {
		t.Fatalf("got %d result(s), want 2: %+v", len(got), got)
	}
	if got[1].CIDR != "8.8.8.0/24" {
		t.Errorf("CIDR = %q, want it trimmed", got[1].CIDR)
	}
}
