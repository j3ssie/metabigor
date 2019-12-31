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
	// var options core.Options
	// options.Input = "103.102.128.0/24"
	raw := core.GetFileContent("/tmp/tau/tl.xml")
	result := ParsingNmap(raw)
	fmt.Println(result)
	if len(result) == 0 {
		t.Errorf("Error RunMasscan")
	}
}
