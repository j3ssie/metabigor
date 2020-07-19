package modules

import (
	"fmt"
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"strings"

	jsoniter "github.com/json-iterator/go"

	"github.com/j3ssie/metabigor/core"
)

// RunMasscan run masscan command and return list of port open
func RunMasscan(input string, options core.Options) []string {
	ports := options.Scan.Ports
	rate := options.Scan.Rate
	if ports == "" {
		ports = "443"
	}

	massOutput := options.Scan.TmpOutput
	tmpFile, _ := ioutil.TempFile(options.Scan.TmpOutput, "masscan-*.txt")
	if massOutput != "" {
		tmpFile, _ = ioutil.TempFile(massOutput, fmt.Sprintf("masscan-%v-*.txt", core.StripPath(input)))
	}
	massOutput = tmpFile.Name()

	masscanCmd := fmt.Sprintf("sudo masscan --rate %v -p %v -oG %v %v", rate, ports, massOutput, input)
	core.DebugF("Execute: %v", masscanCmd)
	command := []string{
		"bash",
		"-c",
		masscanCmd,
	}
	exec.Command(command[0], command[1:]...).CombinedOutput()
	// parse output
	var realResult []string
	result := make(map[string]string)
	if !core.FileExists(massOutput) {
		return realResult
	}
	data := core.GetFileContent(massOutput)
	rawResult := ParsingMasscan(data)
	// get flat output for easily parse to other tools
	if options.Scan.Flat {
		for k, v := range rawResult {
			for _, port := range v {
				realResult = append(realResult, fmt.Sprintf("%v:%v", k, port))
			}
		}
		return realResult
	}

	// group them by host
	for k, v := range rawResult {
		result[k] += fmt.Sprintf("%v", strings.Join(v, ","))
	}

	for k, v := range result {
		realResult = append(realResult, fmt.Sprintf("%v - %v", k, v))
	}

	return realResult
}

// RunNmap run nmap command and return list of port open
func RunNmap(input string, ports string, options core.Options) []string {
	// use nmap as overview scan
	if options.Scan.NmapOverview {
		ports = options.Scan.Ports
	}
	if ports == "" {
		ports = "443"
	}
	nmapOutput := options.Scan.TmpOutput
	tmpFile, _ := ioutil.TempFile(options.Scan.TmpOutput, "nmap-*")
	if nmapOutput != "" {
		tmpFile, _ = ioutil.TempFile(nmapOutput, fmt.Sprintf("nmap-%v-*", core.StripPath(input)))
	}
	nmapOutput = tmpFile.Name()
	nmapCmd := fmt.Sprintf("sudo nmap -sSV -p %v %v -T4 --open -oA %v", ports, input, nmapOutput)
	if options.Scan.NmapScripts != "" {
		nmapCmd = fmt.Sprintf("sudo nmap -sSV -p %v %v -T4 --open --script %v -oA %v", ports, input, options.Scan.NmapScripts, nmapOutput)
	}
	core.DebugF("Execute: %v", nmapCmd)
	command := []string{
		"bash",
		"-c",
		nmapCmd,
	}
	exec.Command(command[0], command[1:]...).CombinedOutput()
	var result []string
	realNmapOutput := nmapOutput + ".xml"
	if !core.FileExists(realNmapOutput) {
		core.ErrorF("Result not found: %v", realNmapOutput)
		return result
	}

	data := core.GetFileContent(realNmapOutput)
	result = ParseNmap(data, options)
	return result
}

// ParseNmap parse nmap XML output
func ParseNmap(raw string, options core.Options) []string {
	var result []string
	var hosts []Host
	if strings.Count(raw, "<address") > 1 {
		hosts = append(hosts, GetHosts(raw)...)
	} else {
		hosts = append(hosts, GetHost(raw))
	}

	for _, host := range hosts {
		//spew.Dump(host)
		if len(host.Ports) <= 0 {
			core.ErrorF("No open port found for %v", host.IPAddress)
			continue
		}
		if options.JsonOutput {
			if data, err := jsoniter.MarshalToString(host); err == nil {
				result = append(result, data)
			}
			continue
		}

		for _, port := range host.Ports {
			info := fmt.Sprintf("%v:%v/%v/%v", host.IPAddress, port.PortID, port.Protocol, port.Service.Product)
			//fmt.Println(info)
			result = append(result, info)
		}
	}
	return result
}

// ParsingMasscan parse result from masscan XML format
func ParsingMasscan(raw string) map[string][]string {
	result := make(map[string][]string)
	data := strings.Split(raw, "\n")

	for _, line := range data {
		if !strings.Contains(line, "Host: ") {
			continue
		}
		rawResult := strings.Split(line, " ")
		ip := rawResult[1]
		port := strings.Split(rawResult[len(rawResult)-1], "/")[0]
		result[ip] = append(result[ip], port)
	}

	return result
}

// RunZmap run masscan command and return list of port open
func RunZmap(inputFile string, port string, options core.Options) []string {
	ports := options.Scan.Ports
	if ports == "" {
		ports = "443"
	}
	zmapOutput := options.Scan.TmpOutput
	tmpFile, _ := ioutil.TempFile(options.Scan.TmpOutput, "out-*.txt")
	if zmapOutput != "" {
		tmpFile, _ = ioutil.TempFile(zmapOutput, fmt.Sprintf("out-%v-*.txt", core.StripPath(filepath.Base(inputFile))))
	}
	zmapOutput = tmpFile.Name()
	zmapCmd := fmt.Sprintf("sudo zmap -p %v -w %v -f 'saddr,sport' -O csv -o %v", port, inputFile, zmapOutput)
	core.DebugF("Execute: %v", zmapCmd)
	command := []string{
		"bash",
		"-c",
		zmapCmd,
	}
	exec.Command(command[0], command[1:]...).CombinedOutput()

	result := ParseZmap(zmapOutput)
	return result
}

// ParseZmap parsse zmap data
func ParseZmap(zmapOutput string) []string {
	data := core.GetFileContent(zmapOutput)
	var result []string
	if strings.TrimSpace(data) == "" {
		return result
	}

	raw := strings.Replace(data, ",", ":", -1)
	raw = strings.Replace(raw, "saddr:sport", "", -1)
	raw = strings.TrimSpace(raw)

	result = strings.Split(raw, "\n")
	return result
}
