package modules

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/PuerkitoBio/goquery"
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
	tmpFile, _ := ioutil.TempFile(os.TempDir(), "masscan-*.txt")
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
	// ports := options.Ports
	if ports == "" {
		ports = "443"
	}
	nmapOutput := options.Scan.TmpOutput
	tmpFile, _ := ioutil.TempFile(os.TempDir(), "nmap-*")
	if nmapOutput != "" {
		tmpFile, _ = ioutil.TempFile(nmapOutput, fmt.Sprintf("nmap-%v-*", core.StripPath(input)))
	}
	nmapOutput = tmpFile.Name()
	nmapCmd := fmt.Sprintf("sudo nmap -sSV -p %v %v -T4 -oA %v", ports, input, nmapOutput)
	if options.Scan.NmapScripts != "" {
		nmapCmd = fmt.Sprintf("sudo nmap -sSV -p %v %v -T4 --script %v -oA %v", ports, input, options.Scan.NmapScripts, nmapOutput)
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
		return result
	}
	// result := ""
	data := core.GetFileContent(realNmapOutput)
	rawResult := ParsingNmap(data)
	for k, v := range rawResult {
		result = append(result, fmt.Sprintf("%v - %v", k, strings.Join(v, ",")))
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

// ParsingMasscanXML parse result from masscan XML format
func ParsingMasscanXML(raw string) map[string][]string {
	result := make(map[string][]string)
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return result
	}

	doc.Find("host").Each(func(i int, s *goquery.Selection) {
		ip, _ := s.Find("address").First().Attr("addr")
		port, _ := s.Find("port").First().Attr("portid")
		result[ip] = append(result[ip], port)
	})

	return result
}

// ParsingNmap parse result from nmap XML format
func ParsingNmap(raw string) map[string][]string {
	result := make(map[string][]string)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err != nil {
		return result
	}
	doc.Find("host").Each(func(i int, h *goquery.Selection) {
		ip, _ := h.Find("address").First().Attr("addr")
		h.Find("port").Each(func(j int, s *goquery.Selection) {
			service, _ := s.Find("service").First().Attr("name")
			product, ok := s.Find("service").First().Attr("product")
			if !ok {
				product = ""
			}
			port, _ := s.Attr("portid")
			info := fmt.Sprintf("%v/%v/%v", port, service, product)
			// fmt.Println(ip, port, service)
			result[ip] = append(result[ip], strings.TrimSpace(info))
		})
	})

	return result
}
