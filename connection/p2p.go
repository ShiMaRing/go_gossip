package connection

import (
	"go_gossip/config"
	"go_gossip/utils"
)

func NewBothServer(configPath string) (*PeerServer, *GossipServer) {
	P2PConfig, logConfig, err := config.LoadConfig(configPath)
	if err != nil {
		panic(err)
	}
	Logger := utils.LogInit(logConfig)
	Logger.Info("config loaded")
	Logger.Info("p2p address", "p2pAddress", P2PConfig.P2PAddress, "apiAddress", P2PConfig.APIAddress)
	peerServer := NewPeerServer(P2PConfig, Logger)
	gossipServer := NewGossipServer(P2PConfig, Logger)
	peerServer.peerTcpManager.gossipServer = gossipServer
	gossipServer.gossipTcpManager.peerServer = peerServer
	return peerServer, gossipServer
}

func CloseServer(peerServer *PeerServer, gossipServer *GossipServer) {
	peerServer.peerTcpManager.Stop()
	gossipServer.gossipTcpManager.Stop()
}
