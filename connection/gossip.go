package connection

import "go_gossip/model"

type GossipServer struct {
	tcpServer *TCPServer
}

func NewGossipServer(name string, ipAddr string) *GossipServer {
	gossipServer := &GossipServer{
		tcpServer: NewTCPServer(name, ipAddr),
	}
	return gossipServer
}

func checkGossipFrame(frame *model.CommonFrame) bool {

	return false
}

func handleGossipFrame(frame *model.CommonFrame) bool {
	if frame == nil {
		return false
	}
	switch frame.Type {
	case model.GOSSIP_ANNOUCE:
		//handle announce message
	case model.GOSSIP_NOTIFY:
		//handle notify message

	}
	return false
}
