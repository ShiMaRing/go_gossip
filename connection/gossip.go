package connection

import "go_gossip/model"

type GossipServer struct {
	tcpServer *TCPServer
}

func NewGossipServer(ipAddr string) *GossipServer {
	gossipServer := &GossipServer{
		tcpServer: NewTCPServer("APIServer", ipAddr),
	}
	gossipServer.tcpServer.HandleFrame = handleGossipFrame
	return gossipServer
}

func (g *GossipServer) Start() {
	g.tcpServer.Start()
}

func handleGossipFrame(frame *model.CommonFrame) bool {
	if frame == nil {
		return false
	}
	switch frame.Type {
	case model.GOSSIP_ANNOUCE:
		//handle the announce message
		announce := &model.GossipAnnounceMessage{}
		if !announce.Unpack(frame.Payload) {
			return false
		}
	case model.GOSSIP_NOTIFY:
		//handle the notify message
		notify := &model.GossipNotifyMessage{}
		if !notify.Unpack(frame.Payload) {
			return false
		}

	case model.GOSSIP_NOTIFICATION:
		//handle the notification message
		notification := &model.GossipNotificationMessage{}
		if !notification.Unpack(frame.Payload) {
			return false
		}

	case model.GOSSIP_VALIDATION:
		//handle the validation message
		validation := &model.GossipValidationMessage{}
		if !validation.Unpack(frame.Payload) {
			return false
		}

	default:
		return false //unknown message type
	}
	return false
}
