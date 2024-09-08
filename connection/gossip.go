package connection

import (
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
	"log/slog"
	"net"
	"sync"
	"time"
)

const MaxSubscriber = 16

type GossipServer struct {
	gossipTcpManager     *TCPConnManager
	subscribedConnection map[uint16][]net.Conn //all connection that subscribe to us
	subscribedLock       sync.RWMutex
	waitReplyList        map[uint16]chan bool
	waitReplyLock        sync.RWMutex
	P2PConfig            *config.GossipConfig
	Logger               *slog.Logger
}

func NewGossipServer(P2PConfig *config.GossipConfig, Logger *slog.Logger) *GossipServer {
	gossipServer := &GossipServer{
		gossipTcpManager: NewTCPManager("APIServer", P2PConfig.APIAddress, P2PConfig, Logger),
	}
	gossipServer.gossipTcpManager.HandleFrame = handleGossipFrame
	gossipServer.subscribedConnection = make(map[uint16][]net.Conn)
	gossipServer.waitReplyList = make(map[uint16]chan bool)
	gossipServer.P2PConfig = P2PConfig
	gossipServer.Logger = Logger
	gossipServer.gossipTcpManager.gossipServer = gossipServer //set the gossip server
	return gossipServer
}

func (g *GossipServer) Start() {
	g.gossipTcpManager.StartServer()
}

func (g *GossipServer) SubscribeConnection(dataType uint16, conn net.Conn) {
	g.subscribedLock.Lock()
	defer g.subscribedLock.Unlock()
	if _, ok := g.subscribedConnection[dataType]; !ok {
		g.subscribedConnection[dataType] = make([]net.Conn, 0)
	}
	//check the connection count
	if len(g.subscribedConnection[dataType]) >= MaxSubscriber {
		//close the connection
		conn.Close()
		return
	}
	g.subscribedConnection[dataType] = append(g.subscribedConnection[dataType], conn)
}

func (g *GossipServer) AskSubscribers(broadcast *model.PeerBroadcastMessage) bool {
	g.subscribedLock.RLock()
	if _, ok := g.subscribedConnection[broadcast.Datatype]; !ok {
		g.Logger.Warn("No subscriber for the data type", "dataType", broadcast.Datatype)
		g.subscribedLock.RUnlock()
		return false
	}
	g.Logger.Debug("Ask the subscribers", "dataType", broadcast.Datatype, "subscriberCount", len(g.subscribedConnection[broadcast.Datatype]))
	//generate a gossip notification message
	notification := &model.GossipNotificationMessage{}
	var randUint16 = utils.GenerateRandomNumber()
	g.waitReplyLock.Lock()
	for g.waitReplyList[randUint16] != nil {
		randUint16 = utils.GenerateRandomNumber()
	}
	g.waitReplyList[randUint16] = make(chan bool, MaxSubscriber) //create a channel to wait for the reply
	g.waitReplyLock.Unlock()

	notification.MessageID = randUint16
	notification.DataType = broadcast.Datatype
	notification.Data = broadcast.Data
	//send the message to all subscribers
	frame := model.MakeCommonFrame(model.GOSSIP_NOTIFICATION, notification.Pack())

	mu := sync.Mutex{}
	successCnt := 0
	wg := sync.WaitGroup{}
	wg.Add(len(g.subscribedConnection[broadcast.Datatype]))
	for _, conn := range g.subscribedConnection[broadcast.Datatype] {
		if conn != nil {
			go func(conn net.Conn) {
				defer wg.Done()
				err := g.gossipTcpManager.SendMessage(conn, frame)
				if err != nil {
					conn.Close()
				}
				mu.Lock()
				successCnt++
				mu.Unlock()
			}(conn)
		}
	}
	g.subscribedLock.RUnlock()

	wg.Wait()

	if successCnt == 0 { //all subscribers are failed
		g.waitReplyLock.Lock()
		close(g.waitReplyList[randUint16])
		delete(g.waitReplyList, randUint16)
		g.waitReplyLock.Unlock()
		return false
	}

	//wait here for the reply
	result := true
	timeOut := time.After(5 * time.Second)
	for i := 0; i < successCnt; i++ {
		select {
		case <-timeOut:
			result = false
			break
		case reply := <-g.waitReplyList[randUint16]:
			if !reply {
				result = false
				break
			}
		}
	}

	g.waitReplyLock.Lock()
	close(g.waitReplyList[randUint16])
	delete(g.waitReplyList, randUint16)
	g.waitReplyLock.Unlock()

	return result
}

func handleGossipFrame(frame *model.CommonFrame, conn net.Conn, tcpManager *TCPConnManager) (bool, error) {
	if frame == nil {
		return false, nil
	}
	gossipServer := tcpManager.gossipServer
	peerServer := tcpManager.peerServer
	peerServer.Logger.Debug("Receive a frame", "frame", frame.ToString(), "from", conn.RemoteAddr().String())
	switch frame.Type {
	case model.GOSSIP_ANNOUCE:
		//handle the announcement message
		announce := &model.GossipAnnounceMessage{}
		if !announce.Unpack(frame.Payload) {
			peerServer.Logger.Error("Unpack the announce message failed", "payload", frame.Payload)
			return false, nil
		}
		peerAnnounce := &model.PeerBroadcastMessage{}
		peerAnnounce.Data = announce.Data
		peerAnnounce.Datatype = announce.DataType
		peerAnnounce.Ttl = announce.TTL
		//generate an id
		peerAnnounce.Id = utils.GenerateUUID()
		remain, _ := peerServer.BroadcastMessage(peerAnnounce)
		if remain > 0 {
			peerServer.Logger.Warn("Send the message to peers", "remain", remain)
		} else {
			peerServer.Logger.Warn("Send the message to all peers")
		}
		return true, nil
	case model.GOSSIP_NOTIFY:
		//handle the notify message
		notify := &model.GossipNotifyMessage{}
		if !notify.Unpack(frame.Payload) {
			peerServer.Logger.Error("Unpack the notify message failed")
			return false, nil
		}
		gossipServer.SubscribeConnection(uint16(notify.DataType), conn)
		return true, nil

	case model.GOSSIP_NOTIFICATION:
		//as a server, we should not receive the notification message
		return false, nil //close the connection

	case model.GOSSIP_VALIDATION:
		//handle the validation message
		validation := &model.GossipValidationMessage{}
		if !validation.Unpack(frame.Payload) {
			return false, nil
		}
		//get the id
		id := validation.MessageID
		gossipServer.waitReplyLock.RLock()
		if gossipServer.waitReplyList[id] != nil { //so it was not close
			gossipServer.waitReplyList[id] <- validation.Validation == 1
		}
		gossipServer.waitReplyLock.RUnlock()
	default:
		return false, nil //unknown message type
	}
	return true, nil
}
