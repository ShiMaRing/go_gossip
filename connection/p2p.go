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
	peerServer := NewPeerServer(P2PConfig, Logger)
	gossipServer := NewGossipServer(P2PConfig, Logger)
	peerServer.peerTcpManager.gossipServer = gossipServer
	gossipServer.gossipTcpManager.peerServer = peerServer
	return peerServer, gossipServer
}

func StartServer(peerServer *PeerServer, gossipServer *GossipServer) {
	peerServer.PeerServerStart()
	gossipServer.Start()
}

func CloseServer(peerServer *PeerServer, gossipServer *GossipServer) {
	peerServer.peerTcpManager.Stop()
	gossipServer.gossipTcpManager.Stop()
}
