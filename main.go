package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/golang/glog"
)

const version = "0.2"

func usage() {
	fmt.Fprintf(os.Stderr, "usage: ./mailts -config example/mailts_config.yml"+
		" -stderrthreshold=[INFO|WARN|FATAL]"+
		" -log_dir=./"+
		" -logtostderr=false\n")
	flag.PrintDefaults()
	os.Exit(2)
}

func main() {
	var configFile string
	var useCron bool

	flag.Usage = usage
	flag.StringVar(&configFile, "config", "example/mailts_config.yml", "Path to config file")
	flag.BoolVar(&useCron, "use_cron", false, "Use Cron")
	flag.Parse()
	flag.Lookup("stderrthreshold").Value.Set("INFO")

	if useCron {
		// TODO add cron
	} else {
		report(configFile)
	}

	os.Exit(1)
}

func report(configFile string) {
	config, err := loadConfig(configFile)
	if err != nil {
		log.Fatalln(err)
	}

	writerDone := make(chan interface{}, 1)
	input := make(chan WriterData, 2000)
	go WriterRoutine(writerDone, input)

	reportWorkers := len(config.Query)
	reportWorkersDone := make(chan interface{}, reportWorkers)

	for i := 0; i < reportWorkers; i++ {
		go createReport(&config.Query[i], reportWorkersDone, input)
	}

	for i := 0; i < reportWorkers; i++ {
		<-reportWorkersDone
	}

	close(input)
	<-writerDone
	glog.Flush()
}