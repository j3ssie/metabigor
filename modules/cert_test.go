package modules

import (
	"fmt"
	"github.com/j3ssie/metabigor/core"
	"testing"
)

func TestParseCertSH(t *testing.T) {
	var options core.Options
	raw := core.GetFileContent("/tmp/tll")
	result := ParseCertSH(raw, options)
	fmt.Println(result)
	if len(result) == 0 {
		t.Errorf("Error TestParseCertSH")
	}
}
