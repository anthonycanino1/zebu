package main

import (
	"github.com/anthonycanino1/zebu/src/zebu"
	"os"
)

func main() {
	zebu.CommandLine(os.Args[1:])
}
