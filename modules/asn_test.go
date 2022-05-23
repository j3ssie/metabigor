package modules

import (
    "fmt"
    "github.com/davecgh/go-spew/spew"
    "inet.af/netaddr"
    "testing"
)

func TestGetLocalAsn(t *testing.T) {
    asnAsnMap, err := GetAsnMap()
    var asns []ASInfo

    ip, err := netaddr.ParseIP("8.8.8.8")
    if err != nil {
        return
    }

    if asn := asnAsnMap.ASofIP(ip); asn.AS != 0 {
        asnNum := asn.AS
        asInfo := asnAsnMap.ASInfo(asnNum)
        spew.Dump(asInfo)
    }

    asnN := asnAsnMap.ASInfo(18144)
    spew.Dump(asnN)

    asnd := asnAsnMap.ASDesc("Google")
    spew.Dump(asnd)

    fmt.Println("------------")
    spew.Dump(asns)

}
