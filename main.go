package main

import (
	"flag"
	"log"
	"strings"
)

var (
	inputFile string
	xxxTags   string
	inputDirs string
)

func init() {
	flag.StringVar(&inputFile, "input", "", "path to input file")
	flag.StringVar(&xxxTags, "XXX_skip", "", "skip tags to inject on XXX fields")
	flag.StringVar(&inputDirs, "dirs", "", "inject tags in all .pb.go files in the specified tilde (~) separated dir list (includes subdirectories)")
	flag.BoolVar(&verbose, "verbose", false, "verbose logging")
	flag.BoolVar(&withClean, "with_clean", false, "remove @inject_tag comment from .pb.go after inject done")
	flag.Parse()
}

func main() {
	var xxxSkipSlice []string
	if len(xxxTags) > 0 {
		xxxSkipSlice = strings.Split(xxxTags, ",")
	}
	if len(inputDirs) > 0 {
		files, dirs, processed := []string{}, strings.Split(inputDirs, "~"), 0
		for _, dir := range dirs {
			files = append(files, getFiles(strings.Trim(dir, " "))...)
		}
		logf("%d .pb.go files to process: \n%s\n", len(files), strings.Join(files, "\n"))
		for _, file := range files {
			if err := process(file, xxxSkipSlice); err != nil {
				log.Fatal(err)
			}
			processed++
		}
		logf("processed %d .pb.go files\n", processed)
		return
	}
	if len(inputFile) == 0 {
		log.Fatal("input file is mandatory")
	}
	if err := process(inputFile, xxxSkipSlice); err != nil {
		log.Fatal(err)
	}
	logf("processed 1 .pb.go file\n")
}
