package modules

import (
    "fmt"
    "testing"

    "github.com/j3ssie/metabigor/core"
)

func TestRunMasscan(t *testing.T) {
    var options core.Options
    options.Input = "103.102.128.0/24"
    result := RunMasscan("103.102.128.0/24", options)
    if len(result) == 0 {
        t.Errorf("Error RunMasscan")
    }
}
func TestParsingNmap(t *testing.T) {
    var options core.Options
    options.Scan.NmapScripts = "vulners.nse"
    // options.Input = "103.102.128.0/24"
    raw := core.GetFileContent("/tmp/testttt/samm.xml")
    result := ParseNmap(raw, options)
    fmt.Println(result)
    if len(result) == 0 {
        t.Errorf("Error RunMasscan")
    }
}

func TestParseMassScan(t *testing.T) {
    raw := core.GetFileContent("/tmp/mtt/mrp")
    result := ParsingMasscan(raw)
    fmt.Println(result)
    if len(result) == 0 {
        t.Errorf("Error ParsingMasscan")
    }
}

func TestParseZmap(t *testing.T) {
    result := ParseZmap("/tmp/nn/p443")
    fmt.Println(result)
    if len(result) == 0 {
        t.Errorf("Error ParsingMasscan")
    }
}
