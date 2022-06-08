package cmd

import (
	"fmt"
	"github.com/j3ssie/metabigor/core"
	"github.com/j3ssie/metabigor/modules"
	jsoniter "github.com/json-iterator/go"
	"github.com/panjf2000/ants"
	"github.com/spf13/cobra"
	"strings"
	"sync"
)

func init() {
	var tldCmd = &cobra.Command{
		Use:     "related",
		Aliases: []string{"tld", "relate"},
		Short:   "Finding more related domains of the target by applying various techniques",
		Long:    core.DESC,
		RunE:    runTLD,
	}
	tldCmd.Flags().StringVarP(&options.Tld.Source, "src", "s", "all", "Source for gathering TLD")
	RootCmd.AddCommand(tldCmd)
}

func runTLD(_ *cobra.Command, _ []string) error {
	var wg sync.WaitGroup
	p, _ := ants.NewPoolWithFunc(options.Concurrency, func(i interface{}) {
		job := i.(string)
		TLDJob(job)
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

func TLDJob(raw string) {
	var results []core.RelatedDomain
	switch options.Tld.Source {
	case "all":
		results = append(results, modules.CrtSH(raw, options)...)
		results = append(results, modules.ReverseWhois(raw, options)...)
		results = append(results, modules.GoogleAnalytic(raw, options)...)
	case "crt", "cert":
		results = append(results, modules.CrtSH(raw, options)...)
	case "whois", "who":
		results = append(results, modules.ReverseWhois(raw, options)...)
	case "ua", "gtm", "google-analytic":
		results = append(results, modules.GoogleAnalytic(raw, options)...)
	}

	for _, item := range results {
		if options.JsonOutput {
			if data, err := jsoniter.MarshalToString(item); err == nil {
				fmt.Println(data)
			}
			continue
		}

		if options.Verbose {
			fmt.Println(item.Output)
		} else {
			fmt.Println(item.Domain)
		}
	}
}
