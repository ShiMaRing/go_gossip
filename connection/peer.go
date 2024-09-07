package connection

import (
	"fmt"
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
	"log/slog"
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
	listLock        sync.RWMutex
	cacheSize       int
	ourself         model.PeerInfo
	connIDMapLock   sync.RWMutex
	wg              sync.WaitGroup
	receivedMessage map[uint64]bool
	revMsgLock      sync.RWMutex
	loggerConfig    *config.LogConfig
	P2PConfig       *config.GossipConfig
	Logger          *slog.Logger
}

func (p *PeerServer) Close() {
	p.peerTcpManager.Stop()
}

func (p *PeerServer) PeerServerStart() {

	go p.peerTcpManager.StartServer() //start the peer server
	//wait for some time
	time.Sleep(time.Second * 1)

	p.wg.Add(len(p.P2PConfig.KnownPeers))
	for _, peer := range p.P2PConfig.KnownPeers {
		go func(peer string) {
			defer p.wg.Done()
			err := p.ConnectToPeer(peer)
			if err != nil {
				p.listLock.Lock()
				delete(p.peerInfoMap, peer)
				p.listLock.Unlock()
				return
			}
			//connect success , fetch the peer info from the peer
			peerInfo, err := p.GetPeerInfo(peer)
			if err != nil {
				return
			}
			//now we get the peer info list from the peer
			//add the peer info to the peer info list
			p.listLock.Lock()
			for _, info := range peerInfo {
				//we dont add ourself
				if info.P2PIP == p.ourself.P2PIP {
					continue
				}
				p2pAddr := utils.ConvertNum2Addr(info.P2PIP, info.P2PPort)
				p.peerInfoMap[p2pAddr] = info
			}
			p.listLock.Unlock()
		}(peer)
	}

	p.wg.Wait()
	p.Logger.Info("connect to known peers success", "number", len(p.P2PConfig.KnownPeers))

	//if we don't have known peers, or all the known peers are not available
	if len(p.P2PConfig.KnownPeers) == 0 || p.peerTcpManager.GetPeerConnectionSize() == 0 {
		//connect to the bootstrapper
		err := p.ConnectToPeer(p.P2PConfig.Bootstrapper)
		if err != nil {
			p.Logger.Error("connect to bootstrapper failed", "error", err)
			panic(err)
		}
		//connect success , fetch the peer info from the peer
		infoList, err := p.GetPeerInfo(p.P2PConfig.Bootstrapper)
		if err != nil {
			p.Logger.Error("get peer info from bootstrapper failed", "error", err)
			panic(err)
		}
		p.AddNewPeerInfos(infoList)
	}

	//in this time , we start peer server success, start a go routine to
	go p.PeerBackgroundTask()

	//then start the gossip server and listen here

}

func (p *PeerServer) AddNewPeerInfos(infoList []model.PeerInfo) {
	//now we get the peer info list from the peer
	//add the peer info to the peer info list
	p.listLock.Lock()
	for _, info := range infoList {
		//we dont add ourself
		if info.P2PIP == p.ourself.P2PIP {
			continue
		}
		p2pAddr := utils.ConvertNum2Addr(info.P2PIP, info.P2PPort)
		p.peerInfoMap[p2pAddr] = info
	}
	//check the peer info size
	currentSize := len(p.peerInfoMap)
	//the peer info now is bigger than the cache size
	toDecrease := currentSize - p.cacheSize
	keyToRemove := make([]string, 0)
	if currentSize > p.cacheSize {
		//randomly remove one peer info
		for key := range p.peerInfoMap {
			if key == p.P2PConfig.P2PAddress { //dont remove ourself
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
		if p.peerTcpManager.GetPeerConnection(key) != nil {
			p.peerTcpManager.ClosePeer(key) //close the connection
		}
		delete(p.peerInfoMap, key)
	}
	p.listLock.Unlock()
}

func (p *PeerServer) PeerBackgroundTask() {
	//in this go routine, we will do the following things
	for true {
		time.Sleep(time.Second * time.Duration(p.P2PConfig.MaintainInterval))
		p.listLock.Lock()
		for addr, _ := range p.peerInfoMap {
			if addr == p.P2PConfig.P2PAddress {
				continue
			}
			//check if this peer is connected
			conn := p.peerTcpManager.GetPeerConnection(addr)
			if conn == nil {
				//connect to the peer
				err := p.ConnectToPeer(addr)
				if err != nil {
					delete(p.peerInfoMap, addr)
				}
				info, err := p.GetPeerInfo(addr)
				if err != nil {
					continue
				}
				p.AddNewPeerInfos(info)
				break
			}
		}
		p.listLock.Unlock()
		randCnt := rand.Intn(100) //have 50% chance to close some connection

		//check the connection size , if the connection size is bigger than the min connection size
		needClose := p.peerTcpManager.GetPeerConnectionSize() > p.P2PConfig.MinConnections
		mustClose := p.peerTcpManager.GetPeerConnectionSize() > p.P2PConfig.MaxConnections

		if (randCnt < 50 && needClose) || mustClose {
			//randomly close one connection
			p.listLock.RLock()
			for addr, _ := range p.peerInfoMap {
				if addr == p.P2PConfig.P2PAddress {
					continue
				}
				//check the connection is already connected
				conn := p.peerTcpManager.GetPeerConnection(addr)
				if conn == nil {
					continue
				}
				//close the connection
				p.peerTcpManager.ClosePeer(addr)
				break
			}
			p.listLock.RUnlock()
		}
	}

}

func NewPeerServer(P2PConfig *config.GossipConfig, Logger *slog.Logger) *PeerServer {
	var err error = nil
	//we confirm the newTcpServer will not fail ,if fail ,we panic
	peerServer := &PeerServer{
		peerTcpManager: NewTCPManager("PeerServer", P2PConfig.P2PAddress, P2PConfig, Logger),
	}
	peerServer.peerTcpManager.HandleFrame = handlePeerFrame
	peerServer.P2PConfig = P2PConfig
	peerServer.cacheSize = P2PConfig.CacheSize
	peerServer.peerInfoMap = make(map[string]model.PeerInfo)
	peerServer.Logger = Logger

	var ourSelf = model.PeerInfo{}
	//fill our self info with config
	ourSelf.P2PIP, ourSelf.P2PPort, err = utils.ConvertAddr2Num(P2PConfig.P2PAddress)
	if err != nil {
		peerServer.Logger.Error("convert address to num failed", "error", err)
		panic(err)
	}
	ourSelf.ApiIP, ourSelf.APIPort, err = utils.ConvertAddr2Num(peerServer.P2PConfig.APIAddress)
	if err != nil {
		peerServer.Logger.Error("convert address to num failed", "error", err)
		panic(err)
	}
	peerServer.ourself = ourSelf
	peerServer.peerInfoMap[peerServer.P2PConfig.P2PAddress] = ourSelf
	//add the know peers to the peer info list
	for _, peer := range peerServer.P2PConfig.KnownPeers {
		p2pIP, port, err := utils.ConvertAddr2Num(peer)
		if err != nil {
			peerServer.Logger.Error("convert address to num failed", "error", err)
			continue
		}
		peerInfo := model.PeerInfo{
			P2PIP:   p2pIP,
			P2PPort: port,
		}
		peerServer.peerInfoMap[peer] = peerInfo //incomplete info
	}
	peerServer.peerTcpManager.peerServer = peerServer //set the peer server
	return peerServer
}

// ConnectToPeer connect to the peer
func (p *PeerServer) ConnectToPeer(ipAddr string) error {
	err := p.peerTcpManager.ConnectToPeer(ipAddr)
	if err != nil {
		p.Logger.Error("%s: connect to peer %s failed", p.peerTcpManager.name, ipAddr)
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
		p.Logger.Error("%s: send request message to peer %s failed", p.peerTcpManager.name, ipAddr)
		return err
	}
	//read the conn for the validation message
	headerBuffer := make([]byte, 4)
	_, err = conn.Read(headerBuffer)
	if err != nil {
		p.Logger.Error(" read header failed", "server", p.peerTcpManager.name)
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
		p.Logger.Error("read payload failed", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return err
	}
	if frame.Type != model.PEER_VALIDATION {
		p.Logger.Error("receive a invalid message type", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("invalid message type")
	}
	validation := &model.PeerValidationMessage{}
	if !validation.Unpack(payloadBuffer) {
		p.Logger.Error("unpack validation message failed", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return fmt.Errorf("unpack validation message failed")
	}
	if validation.Validation == 0 {
		p.Logger.Warn("peer reject the connection", "server", p.peerTcpManager.name, "dest", ipAddr)
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
		p.Logger.Error("%s: connection to %s not found", p.peerTcpManager.name, ipAddr)
		return nil, fmt.Errorf("connection to %s not found", ipAddr)
	}
	//ok ,we have the connection to the peer
	//now we need to send the discovery message to the peer
	discovery := &model.PeerDiscoveryMessage{}
	frame := model.MakeCommonFrame(model.PEER_DISCOVERY, discovery.Pack())
	err := p.peerTcpManager.SendMessage(conn, frame)
	if err != nil {
		p.Logger.Error("%s: send discovery message to peer %s failed", p.peerTcpManager.name, ipAddr)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, err
	}
	//read the conn for the peer info message
	headerBuffer := make([]byte, 4)
	_, err = conn.Read(headerBuffer)
	if err != nil {
		p.Logger.Error("read header failed", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, err
	}
	//parse the header , we only want to get a peer info message
	frame = new(model.CommonFrame)
	frame.ParseHeader(headerBuffer)
	if frame.Type != model.PEER_INFO || frame.Size > model.MAX_DATA_SIZE {
		p.Logger.Error("receive a invalid message type", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, fmt.Errorf("invalid message type")
	}
	//read the payload
	var cnt uint16 = 0
	payloadBuffer := make([]byte, frame.Size)
	for cnt < frame.Size {
		n, err := conn.Read(payloadBuffer[cnt:])
		if err != nil {
			p.Logger.Error("read payload failed", "server", p.peerTcpManager.name)
			defer p.peerTcpManager.ClosePeer(ipAddr)
			break
		}
		cnt += uint16(n)
	}
	peerInfo := &model.PeerInfoMessage{}
	if !peerInfo.Unpack(payloadBuffer) {
		p.Logger.Error("unpack peer info message failed", "server", p.peerTcpManager.name)
		defer p.peerTcpManager.ClosePeer(ipAddr)
		return nil, fmt.Errorf("unpack peer info message failed")
	}
	//ok, we get the peer info message
	//extract them and append them to the peer info list
	return peerInfo.Peers, nil
}

func (p *PeerServer) BroadcastMessage(announce *model.PeerBroadcastMessage) (int, error) {
	//check the ttl
	if announce.Ttl == 1 {
		//no need to broadcast
		return 0, nil
	}
	//else, we need to broadcast the message to all the peer we know
	//decrease the ttl
	frame := model.MakeCommonFrame(model.PEER_BROADCAST, announce.Pack())
	degree := p.P2PConfig.Degree //min connection must bigger than degree
	//we always have some connection to the peer, and the number of connection is bigger than the min connection
	//try our best to broadcast the message
	remain := p.peerTcpManager.BroadcastMessageToPeer(frame, degree)
	toRemove := make([]string, 0)
	//choose the peer to broadcast
	p.listLock.RLock()
	for addr, _ := range p.peerInfoMap {
		//don't broadcast to ourselves
		if addr == p.P2PConfig.P2PAddress {
			continue
		}
		//check the connection is already connected
		conn := p.peerTcpManager.GetPeerConnection(addr)
		if conn != nil {
			continue
		}
		//connect to the peer
		err := p.ConnectToPeer(addr)
		if err != nil {
			toRemove = append(toRemove, addr)
			continue
		}
		//connect success, send the message to the peer
		conn = p.peerTcpManager.GetPeerConnection(addr)
		err = p.peerTcpManager.SendMessage(conn, frame)
		if err != nil {
			//close it
			p.peerTcpManager.ClosePeer(addr)
			continue
		}
		remain--
		if remain == 0 {
			break
		}
	}
	p.listLock.RUnlock()

	p.listLock.Lock()
	for _, addr := range toRemove {
		delete(p.peerInfoMap, addr)
	}
	p.listLock.Unlock()
	return remain, nil //return the remain degree
}

// handlePeerFrame handle the peer frame in server side
func handlePeerFrame(frame *model.CommonFrame, conn net.Conn, tcpManager *TCPConnManager) (bool, error) {
	if frame == nil {
		return false, nil
	}
	peerServer := tcpManager.peerServer
	gossipServer := tcpManager.gossipServer
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
		//check the data type, and broadcast the message to subscriber
		if broadcast.Ttl == 1 {
			return true, nil //no need to broadcast
		}
		//check the id ,if we have received this message, we don't need to broadcast
		peerServer.revMsgLock.Lock()
		if peerServer.receivedMessage[broadcast.Id] {
			return true, nil
		}
		peerServer.receivedMessage[broadcast.Id] = true
		peerServer.revMsgLock.Unlock()
		//ask the gossip server can we spread this message

		allow := gossipServer.AskSubscribers(broadcast)
		if !allow {
			return true, nil
		}
		//broadcast the message to the peer
		broadcast.Ttl--
		remain, _ := peerServer.BroadcastMessage(broadcast)
		if remain != 0 {
			peerServer.Logger.Warn("broadcast message to peer failed", "remain", remain)
		}
		return true, nil

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
		if connCnt >= peerServer.P2PConfig.MaxConnections {
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
					if key == peerServer.P2PConfig.P2PAddress {
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
