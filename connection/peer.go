package connection

import (
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
	"net"
	"sync"
)

// PeerServer store the information of the peer
// offer the method to find the new peer and broadcast the message
type PeerServer struct {
	peerTcpManager *TCPConnManager
	peerInfoList   []model.PeerInfo
	listLock       sync.RWMutex
	cacheSize      int
	ourself        model.PeerInfo
}

func NewPeerServer(ipAddr string) *PeerServer {
	var err error = nil
	//we confirm the newTcpServer will not fail ,if fail ,we panic
	peerServer := &PeerServer{
		peerTcpManager: NewTCPManager("PeerServer", ipAddr),
	}
	peerServer.peerTcpManager.HandleFrame = handlePeerFrame
	peerServer.cacheSize = config.P2PConfig.CacheSize
	peerServer.peerInfoList = make([]model.PeerInfo, 0)
	var ourSelf = model.PeerInfo{}
	//fill our self info with config
	ourSelf.P2PIP, ourSelf.P2PPort, err = utils.ConvertAddr2Num(config.P2PConfig.P2PAddress)
	if err != nil {
		utils.Logger.Error("convert address to num failed", err)
		panic(err)
	}
	ourSelf.ApiIP, ourSelf.APIPort, err = utils.ConvertAddr2Num(config.P2PConfig.APIAddress)
	if err != nil {
		utils.Logger.Error("convert address to num failed", err)
		panic(err)
	}
	peerServer.ourself = ourSelf
	peerServer.peerInfoList = append(peerServer.peerInfoList, ourSelf)
	return peerServer
}

// handlePeerFrame handle the peer frame
func handlePeerFrame(frame *model.CommonFrame, conn net.Conn) (bool, error) {
	if frame == nil {
		return false, nil
	}
	switch frame.Type {
	case model.PEER_DISCOVERY:
		discovery := &model.PeerDiscoveryMessage{}
		if !discovery.Unpack(frame.Payload) {
			return false, nil
		}

	case model.PEER_INFO:
		info := &model.PeerInfoMessage{}
		if !info.Unpack(frame.Payload) {
			return false, nil
		}

	case model.PEER_BROADCAST:
		broadcast := &model.PeerBroadcastMessage{}
		if !broadcast.Unpack(frame.Payload) {
			return false, nil
		}
	case model.PEER_VALIDATION:
		validation := &model.PeerValidationMessage{}
		if !validation.Unpack(frame.Payload) {
			return false, nil

		}
	case model.PEER_REQUEST:
		request := &model.PeerRequestMessage{}
		if !request.Unpack(frame.Payload) {
			return false, nil
		}
		//ok, this time we receive a request message

	default:
		return false, nil //unknown message type
	}
	return false, nil

}
