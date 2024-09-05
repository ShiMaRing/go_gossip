package main

import (
	"go_gossip/config"
	"os"
)

func init() {

}

var args []string

func main() {
	args = os.Args[1:]
	if len(args) < 1 {
		panic("config file path is required")
	}
	//get the cmd line args , the input is the file name of config file
	configPath := args[0]
	_, err := config.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	//start the gossip protocol
}
