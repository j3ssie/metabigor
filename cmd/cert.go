package cmd

import (
	"strings"
	"sync"

	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	"github.com/panjf2000/ants"

	"github.com/spf13/cobra"
)

func init() {
	var certCmd = &cobra.Command{
		Use:     "cert",
		Aliases: []string{"crt", "ctr"},
		Short:   "Certificates search",
		Long:    core.DESC,
		RunE:    runCert,
	}

	certCmd.Flags().BoolVarP(&options.Cert.Clean, "clean", "C", false, "Auto clean the result")
	certCmd.Flags().BoolVarP(&options.Cert.OnlyWildCard, "wildcard", "W", false, "Only get wildcard domain")
	RootCmd.AddCommand(certCmd)
}

func runCert(_ *cobra.Command, _ []string) error {
	options.Timeout = options.Timeout * 3

	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(options.Concurrency, func(i interface{}) {
		job := i.(string)
		searchResult := runCertSearch(job, options)
		StoreData(searchResult, options)
		wg.Done()
	}, ants.WithPreAlloc(true))
	defer p.Release()

	for _, target := range options.Inputs {
		wg.Add(1)
		_ = p.Invoke(strings.TrimSpace(target))
	}

	wg.Wait()
	core.Unique(options.Output)
	return nil
}

func runCertSearch(input string, options core.Options) []string {
	var data []string
	core.BannerF("Searching on crt.sh for: ", input)
	result := modules.CrtSHOrg(input, options)
	data = append(data, result...)
	return data
}
