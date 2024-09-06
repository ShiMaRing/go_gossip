package connection

import (
	"go_gossip/model"
	"net"
)

type GossipServer struct {
	gossipTcpManager *TCPConnManager
	peerServer       *PeerServer
}

func NewGossipServer(ipAddr string) *GossipServer {
	gossipServer := &GossipServer{
		gossipTcpManager: NewTCPManager("APIServer", ipAddr),
	}
	gossipServer.gossipTcpManager.HandleFrame = handleGossipFrame
	return gossipServer
}

func handleGossipFrame(frame *model.CommonFrame, conn net.Conn, tcpManager *TCPConnManager) (bool, error) {
	if frame == nil {
		return false, nil
	}
	switch frame.Type {
	case model.GOSSIP_ANNOUCE:
		//handle the announcement message
		announce := &model.GossipAnnounceMessage{}
		if !announce.Unpack(frame.Payload) {
			return false, nil
		}
	case model.GOSSIP_NOTIFY:
		//handle the notify message
		notify := &model.GossipNotifyMessage{}
		if !notify.Unpack(frame.Payload) {
			return false, nil
		}

	case model.GOSSIP_NOTIFICATION:
		//handle the notification message
		notification := &model.GossipNotificationMessage{}
		if !notification.Unpack(frame.Payload) {
			return false, nil
		}

	case model.GOSSIP_VALIDATION:
		//handle the validation message
		validation := &model.GossipValidationMessage{}
		if !validation.Unpack(frame.Payload) {
			return false, nil
		}

	default:
		return false, nil //unknown message type
	}
	return false, nil
}
