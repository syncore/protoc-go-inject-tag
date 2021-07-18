package main

import (
	"log"
)

var verbose = false

func logf(format string, v ...interface{}) {
	if !verbose {
		return
	}
	log.Printf("protoc-go-inject-tag: "+format, v...)
}
