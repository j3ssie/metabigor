package modules

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/j3ssie/metabigor/core"
	jsoniter "github.com/json-iterator/go"
	"testing"
)

func TestParseNmapXML(t *testing.T) {
	raw := core.GetFileContent("/tmp/nn/vuln.xml")
	nmapRun := ParseNmapXML(raw)
	spew.Dump(nmapRun.Host)
	fmt.Println("------------")
	spew.Dump(nmapRun.Host.Ports)
	//if len(result) == 0 {
	//	t.Errorf("Error ParsingMasscan")
	//}
}

func TestGetHost(t *testing.T) {
	raw := core.GetFileContent("/var/folders/lx/q7xk40_d3vd_wvw5dpdj796r0000gn/T/mtg-log/nmap-77.111.191.237-378284373.xml")
	//raw := core.GetFileContent("/tmp/nn/full.xml")
	host := GetHost(raw)
	spew.Dump(host)

	fmt.Println("------------")

	// output as JSON
	if data, err := jsoniter.MarshalToString(host); err == nil {
		fmt.Println(data)
	}

	if len(host.Ports) > 0 {
		for _, port := range host.Ports {
			info := fmt.Sprintf("%v:%v/%v/%v", host.IPAddress, port.PortID, port.Protocol, port.Service.Product)
			fmt.Println(info)
		}
	}
}

func TestGetHosts(t *testing.T) {
	raw := core.GetFileContent("/tmp/nn/multiple.xml")
	hosts := GetHosts(raw)
	//spew.Dump(host)

	for _, host := range hosts {
		fmt.Println("------------")
		// output as JSON
		if data, err := jsoniter.MarshalToString(host); err == nil {
			fmt.Println(data)
		}

		if len(host.Ports) > 0 {
			for _, port := range host.Ports {
				info := fmt.Sprintf("%v:%v/%v/%v", host.IPAddress, port.PortID, port.Protocol, port.Service.Product)
				fmt.Println(info)
			}
		}
	}

}
