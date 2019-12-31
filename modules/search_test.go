package modules

import (
	"testing"

	"github.com/j3ssie/metabigor/core"
)

func TestFofa(t *testing.T) {
	var options core.Options
	options.Input = "103.102.128.0/24"
	result := RunMasscan("103.102.128.0/24", options)
	if len(result) == 0 {
		t.Errorf("Error RunMasscan")
	}
}
