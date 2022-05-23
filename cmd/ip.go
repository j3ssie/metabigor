package cmd

import (
    "fmt"
    "github.com/j3ssie/metabigor/core"
    "github.com/j3ssie/metabigor/modules"
    jsoniter "github.com/json-iterator/go"
    "github.com/panjf2000/ants"
    "github.com/projectdiscovery/mapcidr"
    "github.com/spf13/cobra"
    "strings"
    "sync"

    "net"
)

var csvOutput = false
var onlyHost = false

func init() {
    var ipCmd = &cobra.Command{
        Use:   "ip",
        Short: "Extract Shodan IPInfo from internetdb.shodan.io",
        Long:  core.DESC,
        RunE:  runIP,
    }
    ipCmd.PersistentFlags().Bool("csv", true, "Show Output as CSV format")
    ipCmd.PersistentFlags().Bool("open", false, "Show Output as format 'IP:Port' only")
    RootCmd.AddCommand(ipCmd)
}

func runIP(cmd *cobra.Command, _ []string) error {
    csvOutput, _ = cmd.Flags().GetBool("csv")
    onlyHost, _ = cmd.Flags().GetBool("open")

    var wg sync.WaitGroup
    p, _ := ants.NewPoolWithFunc(options.Concurrency, func(i interface{}) {
        job := i.(string)
        StartJob(job)
        wg.Done()
    }, ants.WithPreAlloc(true))
    defer p.Release()

    for _, target := range options.Inputs {
        wg.Add(1)
        _ = p.Invoke(strings.TrimSpace(target))
    }

    wg.Wait()
    return nil
}

func StartJob(raw string) {
    _, _, err := net.ParseCIDR(raw)
    if err != nil {
        GetShodanIPInfo(raw)
        return
    }

    if ips, err := mapcidr.IPAddresses(raw); err == nil {
        for _, ip := range ips {
            GetShodanIPInfo(ip)
        }
    }
}

type ShodanIPInfo struct {
    Cpes      []string `json:"cpes"`
    Hostnames []string `json:"hostnames"`
    IP        string   `json:"ip"`
    Ports     []int    `json:"ports"`
    Tags      []string `json:"tags"`
    Vulns     []string `json:"vulns"`
}

func GetShodanIPInfo(IP string) {
    data := modules.InternetDB(IP)
    if data == "" {
        core.ErrorF("No data found for: %s", IP)
        return
    }

    if options.JsonOutput {
        fmt.Println(data)
        return
    }

    var shodanIPInfo ShodanIPInfo
    if ok := jsoniter.Unmarshal([]byte(data), &shodanIPInfo); ok != nil {
        return
    }

    if csvOutput {
        for _, port := range shodanIPInfo.Ports {
            line := fmt.Sprintf("%s:%d", IP, port)
            if onlyHost {
                fmt.Println(line)
                continue
            }

            line = fmt.Sprintf("%s,%s,%s,%s,%s", line, strings.Join(shodanIPInfo.Hostnames, ";"), strings.Join(shodanIPInfo.Tags, ";"), strings.Join(shodanIPInfo.Cpes, ";"), strings.Join(shodanIPInfo.Vulns, ";"))
            fmt.Println(line)
        }
    }
}
