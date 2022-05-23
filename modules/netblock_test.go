package modules

import (
    "testing"

    "github.com/j3ssie/metabigor/core"
)

func TestIPInfo(t *testing.T) {
    var options core.Options
    options.Net.Asn = "AS714"
    result := GetIPInfo(options)
    if len(result) == 0 {
        t.Errorf("Error IPInfo")
    }
}

func TestIPv4Info(t *testing.T) {
    var options core.Options
    options.Net.Asn = "AS6432"
    result := IPv4Info(options)
    if len(result) == 0 {
        t.Errorf("Error IPv4Info")
    }
}

func TestASNBgpDotNet(t *testing.T) {
    var options core.Options
    options.Net.Asn = "AS62830"
    result := ASNBgpDotNet(options)
    if len(result) == 0 {
        t.Errorf("Error TestASNBgpDotNet")
    }
}

func TestASNSpyse(t *testing.T) {
    var options core.Options
    options.Net.Asn = "714"
    result := ASNSpyse(options)
    if len(result) == 0 {
        t.Errorf("Error TestASNSpyse")
    }
}

func TestOrgBgpDotNet(t *testing.T) {
    var options core.Options
    options.Net.Org = "riot"
    result := OrgBgpDotNet(options)
    if len(result) == 0 {
        t.Errorf("Error TestOrgBgpDotNet")
    }
}

func TestASNLookup(t *testing.T) {
    var options core.Options
    options.Net.Org = "riot"
    result := ASNLookup(options)
    if len(result) == 0 {
        t.Errorf("Error ASNLookup")
    }
}

func TestASNFromIP(t *testing.T) {
    var options core.Options
    options.Net.IP = "168.120.1.1"
    result := ASNFromIP(options)
    if len(result) == 0 {
        t.Errorf("Error ASNLookup")
    }
}
