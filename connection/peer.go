package connection

import (
	"fmt"
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
	"math/rand"
	"net"
	"sync"
)

// PeerServer store the information of the peer
// offer the method to find the new peer and broadcast the message
type PeerServer struct {
	peerTcpManager *TCPConnManager
	peerInfoList   []model.PeerInfo //max cache size will control the size of peer info list,
	// only the peer we know both the p2p and gossip address will be added to the
	//peer info list
	listLock      sync.RWMutex
	peerGossipMap map[string]string //key is the p2p addr and value is the gossip addr
	mapLock       sync.RWMutex
	cacheSize     int
	ourself       model.PeerInfo
	gossipServer  *GossipServer
	connIDMapLock sync.RWMutex
}

var peerServer *PeerServer

// PeerServerStart start the peer server, the server will automatically connect to the known peers
// and fetch the peer info from the known peers, util the
func PeerServerStart(ipAddr string) {

}

func NewPeerServer(ipAddr string) {
	var err error = nil
	//we confirm the newTcpServer will not fail ,if fail ,we panic
	peerServer := &PeerServer{
		peerTcpManager: NewTCPManager("PeerServer", ipAddr),
	}
	peerServer.peerTcpManager.HandleFrame = handlePeerFrame
	peerServer.cacheSize = config.P2PConfig.CacheSize
	peerServer.peerInfoList = make([]model.PeerInfo, 0)
	peerServer.peerGossipMap = make(map[string]string)
	peerServer.peerGossipMap[config.P2PConfig.P2PAddress] = config.P2PConfig.APIAddress
	peerServer.peerGossipMap[config.P2PConfig.Bootstrapper] = "" //we don't know the gossip address of the bootstrapper
	for _, peer := range config.P2PConfig.KnownPeers {
		peerServer.peerGossipMap[peer] = "" //we don't know the gossip address of the known peers
	}
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
	peerServer.peerInfoList = append(peerServer.peerInfoList, ourSelf) //add our self info to the peer info list
}

// ConnectToPeer connect to the peer
func (p *PeerServer) ConnectToPeer(ipAddr string) error {
	err := p.peerTcpManager.ConnectToPeer(ipAddr)
	if err != nil {
		utils.Logger.Error("%s: connect to peer %s failed", p.peerTcpManager.name, ipAddr)
		return err
	}
	// we need to let the peer trust us, so we need to send a request message to the peer
	request := &model.PeerRequestMessage{}
	// generate a uint16 rand id
	request.MessageID = uint16(rand.Intn(65536))
	request.PeerInfo = p.ourself
	// send the message to the server
	conn := p.peerTcpManager.GetConnectionByAddr(ipAddr)
	if conn == nil {
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("connection to %s not found", ipAddr)
	}
	frame := model.MakeCommonFrame(model.PEER_REQUEST, request.Pack())
	err = p.peerTcpManager.SendMessage(conn, frame)
	if err != nil {
		defer p.peerTcpManager.ClosePeer(ipAddr)
		utils.Logger.Error("%s: send request message to peer %s failed", p.peerTcpManager.name, ipAddr)
		return err
	}
	//read the conn for the validation message
	headerBuffer := make([]byte, 4)
	_, err = conn.Read(headerBuffer)
	if err != nil {
		utils.Logger.Error("%s: read header failed", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return err
	}
	//parse the header , we only want to get a validation message
	frame = new(model.CommonFrame)
	frame.ParseHeader(headerBuffer)
	//read the payload
	payloadBuffer := make([]byte, frame.Size)
	_, err = conn.Read(payloadBuffer)
	if err != nil {
		utils.Logger.Error("%s: read payload failed", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return err
	}
	if frame.Type != model.PEER_VALIDATION {
		utils.Logger.Error("%s: receive a invalid message type", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("invalid message type")
	}
	validation := &model.PeerValidationMessage{}
	if !validation.Unpack(payloadBuffer) {
		utils.Logger.Error("%s: unpack validation message failed", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("unpack validation message failed")
	}
	if validation.Validation == 0 {
		utils.Logger.Warn("%s: peer %s reject the connection", p.peerTcpManager.name, ipAddr)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("peer reject the connection")
	}
	//ok, the peer accept the connection
	return nil
}

// GetPeerInfo use discovery message to get the peer info
func (p *PeerServer) GetPeerInfo(ipAddr string) ([]model.PeerInfo, error) {
	conn := p.peerTcpManager.GetPeerConnection(ipAddr)
	if conn == nil {
		utils.Logger.Error("%s: connection to %s not found", p.peerTcpManager.name, ipAddr)
		return nil, fmt.Errorf("connection to %s not found", ipAddr)
	}
	//ok ,we have the connection to the peer
	//now we need to send the discovery message to the peer
	discovery := &model.PeerDiscoveryMessage{}
	frame := model.MakeCommonFrame(model.PEER_DISCOVERY, discovery.Pack())
	err := p.peerTcpManager.SendMessage(conn, frame)
	if err != nil {
		utils.Logger.Error("%s: send discovery message to peer %s failed", p.peerTcpManager.name, ipAddr)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, err
	}
	//read the conn for the peer info message
	headerBuffer := make([]byte, 4)
	_, err = conn.Read(headerBuffer)
	if err != nil {
		utils.Logger.Error("%s: read header failed", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, err
	}
	//parse the header , we only want to get a peer info message
	frame = new(model.CommonFrame)
	frame.ParseHeader(headerBuffer)
	if frame.Type != model.PEER_INFO || frame.Size > model.MAX_DATA_SIZE {
		utils.Logger.Error("%s: receive a invalid message type", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, fmt.Errorf("invalid message type")
	}
	//read the payload
	var cnt uint16 = 0
	payloadBuffer := make([]byte, frame.Size)
	for cnt < frame.Size {
		n, err := conn.Read(payloadBuffer[cnt:])
		if err != nil {
			utils.Logger.Error("%s: read payload failed", p.peerTcpManager.name)
			defer p.peerTcpManager.ClosePeer(ipAddr)
			break
		}
		cnt += uint16(n)
	}
	peerInfo := &model.PeerInfoMessage{}
	if !peerInfo.Unpack(payloadBuffer) {
		utils.Logger.Error("%s: unpack peer info message failed", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, fmt.Errorf("unpack peer info message failed")
	}
	//ok, we get the peer info message
	//extract them and append them to the peer info list
	return peerInfo.Peers, nil
}

// handlePeerFrame handle the peer frame in server side
func handlePeerFrame(frame *model.CommonFrame, conn net.Conn, tcpManager *TCPConnManager) (bool, error) {
	if frame == nil {
		return false, nil
	}
	switch frame.Type {
	case model.PEER_DISCOVERY:
		discovery := &model.PeerDiscoveryMessage{}
		if !discovery.Unpack(frame.Payload) {
			return false, nil
		}
		//check the connection is the trusted connection?
		if !tcpManager.IsTrustedConnection(conn) {
			return false, nil
		}
		//ok, this time we receive a discovery message
		//send the peer info to the peer
		peerInfo := &model.PeerInfoMessage{}
		peerServer.listLock.RLock()
		defer peerServer.listLock.RUnlock()
		peerInfo.Cnt = uint16(len(peerServer.peerInfoList))
		//add our self info to the peer info list
		peerInfo.Peers = peerServer.peerInfoList
		newFrame := model.MakeCommonFrame(model.PEER_INFO, peerInfo.Pack())
		err := tcpManager.SendMessage(conn, newFrame) //send the peer info to the peer
		if err != nil {
			return false, err
		}
	case model.PEER_INFO: //as a server ,we don't deal with the info message
		return false, nil

	case model.PEER_BROADCAST:
		broadcast := &model.PeerBroadcastMessage{}
		if !broadcast.Unpack(frame.Payload) {
			return false, nil
		}
		if !tcpManager.IsTrustedConnection(conn) {
			return false, nil
		}
		//ok, this time we receive a broadcast message

	case model.PEER_VALIDATION: //this is the packet for client , as a server ,we don't deal with it
		return false, nil

	case model.PEER_REQUEST:
		request := &model.PeerRequestMessage{}
		if !request.Unpack(frame.Payload) {
			return false, nil
		}
		//ok, this time we receive a request message
		//check the max connection cnt
		var newFrame *model.CommonFrame
		connCnt := tcpManager.GetConnectionCnt()
		if connCnt >= config.P2PConfig.MaxConnections {
			reject := &model.PeerValidationMessage{}
			reject.MessageID = request.MessageID
			reject.Validation = 0 //reject
			newFrame = model.MakeCommonFrame(model.PEER_VALIDATION, reject.Pack())
		} else {
			accept := &model.PeerValidationMessage{}
			accept.MessageID = request.MessageID
			accept.Validation = 1 //accept
			newFrame = model.MakeCommonFrame(model.PEER_VALIDATION, accept.Pack())
			tcpManager.TrustConnection(conn)
			//add the peerInfo to our cache
			peerInfo := request.PeerInfo

			peerServer.listLock.Lock()
			defer peerServer.listLock.Unlock()

			p2pAddr := utils.ConvertNum2Addr(peerInfo.P2PIP, peerInfo.P2PPort)
			apiAddr := utils.ConvertNum2Addr(peerInfo.ApiIP, peerInfo.APIPort)
			if _, ok := peerServer.peerGossipMap[p2pAddr]; !ok {
				peerServer.peerGossipMap[p2pAddr] = apiAddr
				peerServer.peerInfoList = append(peerServer.peerInfoList, peerInfo)
				if len(peerServer.peerInfoList) > peerServer.cacheSize {
					peerServer.peerInfoList = peerServer.peerInfoList[1:] //remove the first element
				}
			} else {
				peerServer.peerGossipMap[p2pAddr] = apiAddr
			}
		}
		err := tcpManager.SendMessage(conn, newFrame) //send the reject or accept message to the peer
		if err != nil {
			return false, err
		}
	default:
		return false, nil //unknown message type
	}
	return true, nil

}
