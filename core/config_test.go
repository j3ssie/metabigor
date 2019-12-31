package core

import (
	"fmt"
	"testing"
)

func TestGetCred(t *testing.T) {
	var options Options
	options.Debug = true
	options.ConfigFile = "~/.metabigor/config.yaml"
	cred := GetCred("fofa", options)
	fmt.Println(cred)
	if cred == "" {
		t.Errorf("Error GetCred")
	}
}
func TestSaveSess(t *testing.T) {
	var options Options
	options.Debug = true
	options.ConfigFile = "~/.metabigor/config.yaml"
	cred := SaveSess("fofa", "_fofapro_ars_session=e059371a4bad81d3c47ce6290240aee4", options)
	fmt.Println(cred)
	if cred == "" {
		t.Errorf("Error SaveSess")
	}
}
