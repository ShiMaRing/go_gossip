package main

import (
	"go_gossip/connection"
	"os"
)

var args = os.Args[1:]

// start a server
// the parameter is the config file path, will panic when the config file is not found
func main() {
	if len(args) < 1 {
		panic("config file path is required")
	}
	configPath := args[0]
	currentPath, _ := os.Getwd()
	println("current path", currentPath)
	println("config path", configPath)

	peerServer, gossipServer := connection.NewBothServer(configPath)
	peerServer.PeerServerStart()
	gossipServer.Start()
	defer connection.CloseServer(peerServer, gossipServer)
}
