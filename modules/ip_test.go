package modules

import (
	"fmt"
	"github.com/j3ssie/metabigor/core"
	"testing"
)

func TestParseOnyphe(t *testing.T) {
	var options core.Options
	raw := core.GetFileContent("/tmp/testttt/ony.html")
	result := ParseOnyphe(raw, options)
	fmt.Println(result)
	if len(result) == 0 {
		t.Errorf("Error parseOnyphe")
	}

}
