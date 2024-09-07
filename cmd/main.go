package main

import (
	"os"
)

var args = os.Args[1:]

func main() {
	if len(args) < 1 {
		panic("config file path is required")
	}

}
