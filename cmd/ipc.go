package cmd

import (
    "fmt"
    "github.com/j3ssie/metabigor/core"
    "github.com/j3ssie/metabigor/modules"
    jsoniter "github.com/json-iterator/go"
    "github.com/spf13/cobra"
    "inet.af/netaddr"
    "os"
    "sort"
    "strings"
)

func init() {
    var netCmd = &cobra.Command{
        Use:   "ipc",
        Short: "Summary about IP list (powered by @thebl4ckturtle)",
        Long:  core.DESC,
        RunE:  runIPC,
    }
    RootCmd.AddCommand(netCmd)
}

func runIPC(_ *cobra.Command, _ []string) error {
    // prepare input
    var inputs []string
    if strings.Contains(options.Input, "\n") {
        inputs = strings.Split(options.Input, "\n")
    } else {
        inputs = append(inputs, options.Input)
    }

    var err error
    ASNMap, err = modules.GetAsnMap()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error to generate asn info")
        os.Exit(-1)
    }

    summary := map[string]*modules.ASInfo{}
    groupByAsn := map[int][]*modules.ASInfo{}

    // do real stuff here
    for _, job := range inputs {
        ip, err := netaddr.ParseIP(strings.TrimSpace(job))
        if err != nil {
            continue
        }
        if asn := ASNMap.ASofIP(ip); asn.AS != 0 {
            if result, ok := summary[asn.CIDR]; ok {
                result.Amount++
                continue
            } else {
                summary[asn.CIDR] = &modules.ASInfo{
                    Amount:      1,
                    Number:      asn.AS,
                    CountryCode: ASNMap.ASCountry(asn.AS),
                    Description: ASNMap.ASName(asn.AS),
                    CIDR:        asn.CIDR,
                }
            }
        }
    }

    for _, result := range summary {
        if _, ok := groupByAsn[result.Number]; ok {
            groupByAsn[result.Number] = append(groupByAsn[result.Number], result)
        } else {
            groupByAsn[result.Number] = []*modules.ASInfo{}
            groupByAsn[result.Number] = append(groupByAsn[result.Number], result)
        }
    }

    // do summary here
    var groupbyCIDR []AsnSummaryByCIDR
    for asnNumber, asnInfos := range groupByAsn {
        //fmt.Printf("AS: %d - %s\n", asnNumber, asnInfos[0].Description)

        sort.Slice(asnInfos, func(i, j int) bool {
            return asnInfos[i].Amount > asnInfos[j].Amount
        })

        //asnSum.Count +=
        for _, as := range asnInfos {
            //fmt.Printf("\t%-16s\t%-4d IPs\n", as.CIDR, as.Amount)

            var asnSum AsnSummaryByCIDR
            asnSum.CIDR = as.CIDR
            asnSum.Count = as.Amount
            asnSum.Number = asnNumber
            asnSum.CountryCode = as.CountryCode
            asnSum.Description = asnInfos[0].Description
            groupbyCIDR = append(groupbyCIDR, asnSum)
        }
    }

    // print the output here
    var contents []string
    for _, asnSum := range groupbyCIDR {
        if options.JsonOutput {
            if data, err := jsoniter.MarshalToString(asnSum); err == nil {
                contents = append(contents, data)
            }
            continue
        }

        data := fmt.Sprintf("%d - %s - %d", asnSum.Number, asnSum.CIDR, asnSum.Count)
        contents = append(contents, data)
    }
    StoreData(contents, options)

    if !core.FileExists(options.Output) {
        core.ErrorF("No data found")
    }
    return nil
}

type AsnSummaryByCIDR struct {
    Number      int
    Description string
    CountryCode string
    CIDR        string
    Count       int
}
