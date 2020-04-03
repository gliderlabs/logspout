package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/gliderlabs/logspout/cfg"
	"github.com/gliderlabs/logspout/router"
)

// Version is the running version of logspout
var Version string

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Printf("%s\n", Version)
		os.Exit(0)
	}

	log.Printf("# logspout %s by gliderlabs\n", Version)
	log.Printf("# adapters: %s\n", strings.Join(router.AdapterFactories.Names(), " "))
	log.Printf("# options : ")
	if d := cfg.GetEnvDefault("DEBUG", ""); d != "" {
		log.Printf("debug:%s\n", d)
	}
	if b := cfg.GetEnvDefault("BACKLOG", ""); b != "" {
		log.Printf("backlog:%s\n", b)
	}
	log.Printf("persist:%s\n", cfg.GetEnvDefault("ROUTESPATH", "/mnt/routes"))

	var jobs []string
	for _, job := range router.Jobs.All() {
		if err := job.Setup(); err != nil {
			log.Printf("!! %v\n", err)
			os.Exit(1)
		}
		if job.Name() != "" {
			jobs = append(jobs, job.Name())
		}
	}
	log.Printf("# jobs    : %s\n", strings.Join(jobs, " "))

	routes, _ := router.Routes.GetAll()
	if len(routes) > 0 {
		log.Println("# routes  :")
		w := new(tabwriter.Writer)
		w.Init(os.Stdout, 0, 8, 0, '\t', 0)
		fmt.Fprintln(w, "#   ADAPTER\tADDRESS\tCONTAINERS\tSOURCES\tOPTIONS") //nolint:errcheck
		for _, route := range routes {
			fmt.Fprintf(w, "#   %s\t%s\t%s\t%s\t%s\n",
				route.Adapter,
				route.Address,
				route.FilterID+route.FilterName+strings.Join(route.FilterLabels, ","),
				strings.Join(route.FilterSources, ","),
				route.Options)
		}
		w.Flush()
	} else {
		log.Println("# routes  : none")
	}

	for _, job := range router.Jobs.All() {
		job := job
		go func() {
			log.Fatalf("%s ended: %s", job.Name(), job.Run())
		}()
	}

	select {}
}
