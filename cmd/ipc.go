package cmd

import (
	"fmt"

	"github.com/j3ssie/metabigor/core"
	jsoniter "github.com/json-iterator/go"
	asnmap "github.com/projectdiscovery/asnmap/libs"
	"github.com/spf13/cobra"
)

func init() {
	var netCmd = &cobra.Command{
		Use:   "ipc",
		Short: "Summary about IP list",
		Long:  core.DESC,
		RunE:  runIPC,
	}
	RootCmd.AddCommand(netCmd)
}

var ASNClient *asnmap.Client

func runIPC(_ *cobra.Command, _ []string) error {
	ASNClient, err := asnmap.NewClient()
	if err != nil {
		core.ErrorF("Unable to init asnmap client: %v", err)
		return err
	}

	asnCount := make(map[string]int)
	asnGroupByCIDR := map[string]AsnSummaryByCIDR{}
	asnSums := []AsnSummaryByCIDR{}

	for _, input := range options.Inputs {
		item, err := ASNClient.GetData(input)
		if err != nil {
			continue
		}

		listOfCIDR, err := asnmap.GetCIDR(item)
		if err != nil {
			continue
		}
		for _, cidr := range listOfCIDR {
			asnCount[cidr.String()]++

			asnGroupByCIDR[cidr.String()] = AsnSummaryByCIDR{
				Number:      item[0].ASN,
				Description: item[0].Org,
				CountryCode: item[0].Country,
			}
		}

	}

	for cidr, count := range asnCount {
		asnInfo := asnGroupByCIDR[cidr]
		asnSums = append(asnSums, AsnSummaryByCIDR{
			CIDR:        cidr,
			Count:       count,
			Number:      asnInfo.Number,
			Description: asnInfo.Description,
			CountryCode: asnInfo.CountryCode,
		})

	}

	// print the output here
	var contents []string
	for _, asnSum := range asnSums {
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
	return nil
}

type AsnSummaryByCIDR struct {
	Number      int
	Description string
	CountryCode string
	CIDR        string
	Count       int
}
