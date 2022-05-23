package modules

import (
    "fmt"
    "github.com/j3ssie/metabigor/core"
    "testing"
)

func TestParseOnyphe(t *testing.T) {
    var options core.Options
    options.Verbose = true
    raw := core.GetFileContent("/tmp/testttt/ony.html")
    result := ParseOnyphe(raw)
    fmt.Println(result)
    if len(result) == 0 {
        t.Errorf("Error parseOnyphe")
    }
}

func TestParseShodan(t *testing.T) {
    var options core.Options
    options.Verbose = true
    raw := core.GetFileContent("/tmp/testttt/shodan.html")
    result := ParseShodan(raw)
    fmt.Println(result)
    if len(result) == 0 {
        t.Errorf("Error parseOnyphe")
    }
}
