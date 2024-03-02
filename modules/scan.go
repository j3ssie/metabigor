package modules

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/thoas/go-funk"

	jsoniter "github.com/json-iterator/go"

	"github.com/j3ssie/metabigor/core"
)

// CurrentUser get current user
func CurrentUser() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	username := u.Username
	return username
}

// RunRustScan run masscan command and return list of port open
func RunRustScan(input string, options core.Options) (results []string) {
	//  rustscan --timeout 3000 -b {{.rateRustScan}} --scripts None --range {{.ports}} -a {{.inputFile}} -g >> {{.Output}}/portscan/raw-open-ports.txt"
	tmpOutput := options.Scan.TmpOutput
	tmpFile, _ := os.CreateTemp(options.Scan.TmpOutput, "rustscan-*.txt")
	if tmpOutput != "" {
		tmpFile, _ = os.CreateTemp(tmpOutput, fmt.Sprintf("rustscan-%v-*.txt", core.StripPath(input)))
	}
	tmpOutput = tmpFile.Name()

	ports := fmt.Sprintf("--range %v", options.Scan.Ports)
	if strings.Contains(options.Scan.Ports, ",") || !strings.Contains(options.Scan.Ports, "-") {
		ports = fmt.Sprintf("--ports %v", options.Scan.Ports)
	}

	prefix := fmt.Sprintf("rustscan -b %v", options.Scan.Rate)
	if options.Scan.Timeout != "" {
		prefix += fmt.Sprintf(" --timeout %v", options.Scan.Timeout)
	} else {
		prefix += " --timeout 3000 "
	}
	if options.Scan.Retry != "" {
		prefix += fmt.Sprintf(" --tries %v", options.Scan.Retry)
	}

	rustscanCmd := fmt.Sprintf("%v %v --scripts None -a %v -g >> %v", prefix, ports, input, tmpOutput)
	runOSCommand(rustscanCmd)

	core.InforF("Parsing result from: %v", tmpOutput)
	data := core.GetFileContent(tmpOutput)

	if options.PipeTheOutput {
		fmt.Printf(data)
		return results
	}

	var content []string
	if strings.Contains(data, "\n") {
		content = strings.Split(data, "\n")
	}

	for _, line := range content {
		// 1.2.3.4 -> [80,80,443,443]
		if !strings.Contains(line, " -> ") {
			continue
		}

		ip := strings.Split(line, " -> ")[0]
		rPorts := strings.Split(line, " -> ")[1]
		rPorts = rPorts[1 : len(rPorts)-1]

		if !strings.Contains(rPorts, ",") {
			results = append(results, fmt.Sprintf("%s:%s", ip, rPorts))
		}

		ports := strings.Split(rPorts, ",")
		ports = funk.UniqString(ports)
		for _, port := range ports {
			results = append(results, fmt.Sprintf("%s:%s", ip, port))
		}
	}

	return results
}

func RunMasscan(input string, options core.Options) []string {
	ports := options.Scan.Ports
	rate := options.Scan.Rate
	if ports == "" {
		ports = "443"
	}

	massOutput := options.Scan.TmpOutput
	tmpFile, _ := os.CreateTemp(options.Scan.TmpOutput, "masscan-*.txt")
	if massOutput != "" {
		tmpFile, _ = os.CreateTemp(massOutput, fmt.Sprintf("masscan-%v-*.txt", core.StripPath(input)))
	}
	massOutput = tmpFile.Name()

	masscanCmd := fmt.Sprintf("masscan --rate %v -p %v -oG %v %v", rate, ports, massOutput, input)
	if options.Scan.All {
		masscanCmd = fmt.Sprintf("masscan --rate %v -p %v -oG %v -iL %v", rate, ports, massOutput, input)
	}
	if CurrentUser() != "root" {
		masscanCmd = "sudo " + masscanCmd
	}
	runOSCommand(masscanCmd)

	// parse output
	var realResult []string
	result := make(map[string]string)
	if !core.FileExists(massOutput) {
		core.ErrorF("Output not found: %v", massOutput)
		return realResult
	}
	core.InforF("Parsing result from: %v", massOutput)
	data := core.GetFileContent(massOutput)
	rawResult := ParsingMasscan(data)

	if len(rawResult) == 0 {
		core.ErrorF("Output not found: %v", massOutput)
	}

	// get flat output for easily parse to other tools
	if options.Scan.Flat {
		for k, v := range rawResult {
			for _, port := range v {
				info := fmt.Sprintf("%v:%v", k, port)
				realResult = append(realResult, info)
			}
		}
		return realResult
	}

	// group them by host in verbose mode
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
	tmpFile, _ := os.CreateTemp(options.Scan.TmpOutput, "nmap-*")
	if nmapOutput != "" {
		tmpFile, _ = os.CreateTemp(nmapOutput, fmt.Sprintf("nmap-%v-*", core.StripPath(input)))
	}
	nmapOutput = tmpFile.Name()

	// build nmap command
	if options.Scan.All {
		options.Scan.NmapTemplate = "nmap -sSV -p {{.ports}} -iL {{.input}} {{.script}} -T4 --open -oA {{.output}}"
	}
	nmapCommand := make(map[string]string)
	nmapCommand["output"] = nmapOutput
	nmapCommand["ports"] = ports
	nmapCommand["input"] = input
	if options.Scan.NmapScripts != "" {
		nmapCommand["script"] = fmt.Sprintf("--script %v", options.Scan.NmapScripts)
	} else {
		nmapCommand["script"] = ""
	}
	nmapCmd := ResolveData(options.Scan.NmapTemplate, nmapCommand)
	if CurrentUser() != "root" {
		nmapCmd = "sudo " + nmapCmd
	}

	//
	//nmapCmd := fmt.Sprintf("sudo nmap -sSV -p %v %v -T4 --open -oA %v", ports, input, nmapOutput)
	//if options.Scan.NmapScripts != "" {
	//	nmapCmd = fmt.Sprintf("sudo nmap -sSV -p %v %v --script %v -T4 --open -oA %v", ports, input, options.Scan.NmapScripts, nmapOutput)
	//}

	runOSCommand(nmapCmd)
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
			result = append(result, info)
		}
	}
	return result
}

// ParsingMasscan parse result from masscan XML format
func ParsingMasscan(raw string) map[string][]string {
	result := make(map[string][]string)
	data := strings.Split(raw, "\n")
	if len(data) == 0 {
		core.ErrorF("Invalid Masscan data")
		return result
	}

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
	tmpFile, _ := os.CreateTemp(options.Scan.TmpOutput, "out-*.txt")
	if zmapOutput != "" {
		tmpFile, _ = os.CreateTemp(zmapOutput, fmt.Sprintf("out-%v-*.txt", core.StripPath(filepath.Base(inputFile))))
	}
	zmapOutput = tmpFile.Name()
	zmapCmd := fmt.Sprintf("zmap -p %v -w %v -f 'saddr,sport' -O csv -o %v", port, inputFile, zmapOutput)
	if CurrentUser() != "root" {
		zmapCmd = "sudo " + zmapCmd
	}
	runOSCommand(zmapCmd)

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

// ResolveData resolve template from signature file
func ResolveData(format string, data map[string]string) string {
	t := template.Must(template.New("").Parse(format))
	buf := &bytes.Buffer{}
	err := t.Execute(buf, data)
	if err != nil {
		return format
	}
	return buf.String()
}

func runOSCommand(cmd string) {
	core.DebugF("Execute: %v", cmd)
	command := []string{
		"bash",
		"-c",
		cmd,
	}
	exec.Command(command[0], command[1:]...).CombinedOutput()
}
