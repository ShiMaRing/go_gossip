package connection

import (
	"fmt"
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
	"math/rand"
	"net"
	"sync"
	"time"
)

// PeerServer store the information of the peer
// offer the method to find the new peer and broadcast the message
type PeerServer struct {
	peerTcpManager *TCPConnManager
	peerInfoMap    map[string]model.PeerInfo //max cache size will control the size of peer info list,
	// only the peer we know both the p2p and gossip address will be added to the
	//peer info list
	listLock      sync.RWMutex
	mapLock       sync.RWMutex
	cacheSize     int
	ourself       model.PeerInfo
	gossipServer  *GossipServer
	connIDMapLock sync.RWMutex
	wg            sync.WaitGroup
}

var peerServer *PeerServer

// PeerServerStart start the peer server, the server will automatically connect to the known peers
// and fetch the peer info from the known peers, util the
func PeerServerStart(ipAddr string) {
	//min connection num should bigger than degree
	//so we need to discovery the peer info first, size should more than min connection num
	//when gossip want to connect the degree num of peer, try to use the current open connection first
	//if the current open connection is not enough, we will try to connect to the new peer in the peer info list
	NewPeerServer(ipAddr) //must success , because we will panic when meet error

	go peerServer.StartPeerServer() //start the peer server

	//wait for some time
	time.Sleep(time.Second * 1)

	peerServer.wg.Add(len(config.P2PConfig.KnownPeers))
	for _, peer := range config.P2PConfig.KnownPeers {
		go func(peer string) {
			defer peerServer.wg.Done()
			err := peerServer.ConnectToPeer(peer)
			if err != nil {
				peerServer.listLock.Lock()
				delete(peerServer.peerInfoMap, peer)
				peerServer.listLock.Unlock()
				return
			}
			//connect success , fetch the peer info from the peer
			peerInfo, err := peerServer.GetPeerInfo(peer)
			if err != nil {
				return
			}
			//now we get the peer info list from the peer
			//add the peer info to the peer info list
			peerServer.listLock.Lock()
			for _, info := range peerInfo {
				//we dont add ourself
				if info.P2PIP == peerServer.ourself.P2PIP {
					continue
				}
				p2pAddr := utils.ConvertNum2Addr(info.P2PIP, info.P2PPort)
				peerServer.peerInfoMap[p2pAddr] = info
			}
			peerServer.listLock.Unlock()
		}(peer)
	}
	peerServer.wg.Wait()

	//if we don't have known peers, or all the known peers are not available
	if len(config.P2PConfig.KnownPeers) == 0 || peerServer.peerTcpManager.GetPeerConnectionSize() == 0 {
		//connect to the bootstrapper
		err := peerServer.ConnectToPeer(config.P2PConfig.Bootstrapper)
		if err != nil {
			utils.Logger.Error("connect to bootstrapper failed", err)
			panic(err)
		}
		//connect success , fetch the peer info from the peer
		infoList, err := peerServer.GetPeerInfo(config.P2PConfig.Bootstrapper)
		if err != nil {
			utils.Logger.Error("get peer info from bootstrapper failed", err)
			panic(err)
		}
		peerServer.AddNewPeerInfos(infoList)
	}

	//in this time , we start peer server success, start a go routine to
	go PeerBackgroundTask()

	//then start the gossip server and listen here

}

func (p *PeerServer) AddNewPeerInfos(infoList []model.PeerInfo) {
	//now we get the peer info list from the peer
	//add the peer info to the peer info list
	peerServer.listLock.Lock()
	for _, info := range infoList {
		//we dont add ourself
		if info.P2PIP == peerServer.ourself.P2PIP {
			continue
		}
		p2pAddr := utils.ConvertNum2Addr(info.P2PIP, info.P2PPort)
		peerServer.peerInfoMap[p2pAddr] = info
	}
	//check the peer info size
	currentSize := len(peerServer.peerInfoMap)
	//the peer info now is bigger than the cache size
	toDecrease := currentSize - peerServer.cacheSize
	keyToRemove := make([]string, 0)
	if currentSize > peerServer.cacheSize {
		//randomly remove one peer info
		for key := range peerServer.peerInfoMap {
			if key == config.P2PConfig.P2PAddress { //dont remove ourself
				continue
			}
			keyToRemove = append(keyToRemove, key)
			toDecrease--
			if toDecrease == 0 {
				break
			}
		}
	}
	for _, key := range keyToRemove {
		//before we delete the peer info, we need to check whether the peer is connected
		if peerServer.peerTcpManager.GetPeerConnection(key) != nil {
			peerServer.peerTcpManager.ClosePeer(key) //close the connection
		}
		delete(peerServer.peerInfoMap, key)
	}
	peerServer.listLock.Unlock()
}

func PeerBackgroundTask() {
	//in this go routine, we will do the following things
	for true {
		time.Sleep(time.Second * time.Duration(config.P2PConfig.MaintainInterval))
		connectionNum := peerServer.peerTcpManager.GetPeerConnectionSize()
		if connectionNum < config.P2PConfig.MinConnections {
			//use the peer info list to connect to the new peer
			peerServer.listLock.RLock()
			for addr, _ := range peerServer.peerInfoMap {
				if addr == config.P2PConfig.P2PAddress {
					continue
				}
				//check the connection is already connected
				if peerServer.peerTcpManager.GetConnectionByAddr(addr) != nil {
					continue
				}
				//connect to the peer
				err := peerServer.ConnectToPeer(addr)
				if err != nil {
					peerServer.listLock.Lock()
					delete(peerServer.peerInfoMap, addr)
					peerServer.listLock.Unlock()
				}
				info, err := peerServer.GetPeerInfo(addr)
				if err != nil {
					continue
				}
				peerServer.AddNewPeerInfos(info)
				//check whether the connection num is enough
				connectionNum = peerServer.peerTcpManager.GetPeerConnectionSize()
				if connectionNum >= config.P2PConfig.MinConnections {
					break
				}
			}
		} else if connectionNum > config.P2PConfig.MaxConnections {
			//randomly close some connection
			toClose := connectionNum - config.P2PConfig.MaxConnections
			peerServer.listLock.RLock()
			for addr, _ := range peerServer.peerInfoMap {
				if addr == config.P2PConfig.P2PAddress {
					continue
				}
				//check the connection is already connected
				conn := peerServer.peerTcpManager.GetPeerConnection(addr)
				if conn == nil {
					continue
				}
				//close the connection
				peerServer.peerTcpManager.ClosePeer(addr)
				toClose--
				if toClose == 0 {
					break
				}
			}
		} else {
			for addr, _ := range peerServer.peerInfoMap {
				if addr == config.P2PConfig.P2PAddress {
					continue
				}
				//check if this peer is connected
				conn := peerServer.peerTcpManager.GetPeerConnection(addr)
				if conn == nil {
					//connect to the peer
					err := peerServer.ConnectToPeer(addr)
					if err != nil {
						peerServer.listLock.Lock()
						delete(peerServer.peerInfoMap, addr)
						peerServer.listLock.Unlock()
					}
					info, err := peerServer.GetPeerInfo(addr)
					if err != nil {
						continue
					}
					peerServer.AddNewPeerInfos(info)
					break
				}
			}
			randCnt := rand.Intn(100) //have 50% chance to close some connection
			if randCnt < 50 {
				//randomly close some connection
				peerServer.listLock.RLock()
				for addr, _ := range peerServer.peerInfoMap {
					if addr == config.P2PConfig.P2PAddress {
						continue
					}
					//check the connection is already connected
					conn := peerServer.peerTcpManager.GetPeerConnection(addr)
					if conn == nil {
						continue
					}
					//close the connection
					peerServer.peerTcpManager.ClosePeer(addr)
					break
				}
			}

		}

	}

}

func NewPeerServer(ipAddr string) {
	var err error = nil
	//we confirm the newTcpServer will not fail ,if fail ,we panic
	peerServer := &PeerServer{
		peerTcpManager: NewTCPManager("PeerServer", ipAddr),
	}
	peerServer.peerTcpManager.HandleFrame = handlePeerFrame
	peerServer.cacheSize = config.P2PConfig.CacheSize
	peerServer.peerInfoMap = make(map[string]model.PeerInfo)
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
	peerServer.peerInfoMap[config.P2PConfig.P2PAddress] = ourSelf
	//add the know peers to the peer info list
	for _, peer := range config.P2PConfig.KnownPeers {
		p2pIP, port, err := utils.ConvertAddr2Num(peer)
		if err != nil {
			utils.Logger.Error("convert address to num failed", err)
			continue
		}
		peerInfo := model.PeerInfo{
			P2PIP:   p2pIP,
			P2PPort: port,
		}
		peerServer.peerInfoMap[peer] = peerInfo //incomplete info
	}
}

func (p *PeerServer) StartPeerServer() {
	p.peerTcpManager.StartServer()
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

// GetPeerInfo use discovery message to get the peer info, only finish this,
// we can control this connection in peer info map
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
		peerInfo.Cnt = uint16(len(peerServer.peerInfoMap))
		//add our self info to the peer info list
		peerInfo.Peers = make([]model.PeerInfo, peerInfo.Cnt)
		for _, info := range peerServer.peerInfoMap {
			peerInfo.Peers = append(peerInfo.Peers, info)
		}
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
			peerServer.peerInfoMap[p2pAddr] = peerInfo //add the peer info to the peer info list because we now all the info of the peer
			//check the peer info size
			if len(peerServer.peerInfoMap) > peerServer.cacheSize {
				//randomly remove one peer info
				for key := range peerServer.peerInfoMap {
					if key == config.P2PConfig.P2PAddress {
						continue
					}
					delete(peerServer.peerInfoMap, key)
					break
				}
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
