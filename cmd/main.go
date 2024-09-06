package main

import (
	"go_gossip/config"
	"go_gossip/utils"
	"os"
)

var args = os.Args[1:]

func main() {
	if len(args) < 1 {
		panic("config file path is required")
	}
	//get the cmd line args , the input is the file name of config file
	configPath := args[0]
	_, err := config.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	utils.LogInit()
	//start the gossip protocol
}
