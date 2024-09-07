package connection

import (
	"go_gossip/config"
	"go_gossip/model"
	"io"
	"log/slog"
	"net"
	"sync"
)

const MaxFrameBuffer = 16

// TCPConnManager Gossip The gossip protocol implementation
type TCPConnManager struct {
	name               string
	ipAddr             string
	HandleFrame        func(*model.CommonFrame, net.Conn, *TCPConnManager) (bool, error)
	server             net.Listener
	closeFlag          bool
	connectionCnt      int                 //current connection count
	handlingConnection map[string]net.Conn //all connection that connect to us
	trustedConnection  map[string]bool     //is this connection trusted ?

	openingConnection map[string]net.Conn //all connection that we connect to, tcp manager will keep this connection
	connRWLock        sync.RWMutex        //lock for the handlingConnection
	cntRWLock         sync.RWMutex
	closeFlagMutex    sync.Mutex

	maxConnCnt   int
	minConnCnt   int
	peerServer   *PeerServer
	gossipServer *GossipServer

	P2PConfig *config.GossipConfig
	Logger    *slog.Logger
}

// NewTCPManager Base on the config apiAddress to create a gossip server
func NewTCPManager(name string, ipAddr string, P2PConfig *config.GossipConfig, Logger *slog.Logger) *TCPConnManager {
	var tcpManager = &TCPConnManager{}
	tcpManager.ipAddr = ipAddr
	tcpManager.name = name
	tcpManager.Logger = Logger
	tcpManager.P2PConfig = P2PConfig
	tcpManager.minConnCnt = tcpManager.P2PConfig.MinConnections
	tcpManager.maxConnCnt = tcpManager.P2PConfig.MaxConnections
	listener, err := net.Listen("tcp", tcpManager.ipAddr)
	if err != nil { // when we start gossip server fail,just panic
		tcpManager.Logger.Error("%s: api server listen failed", tcpManager.name, err)
		panic(err)
	} //now the gossip server is listening
	tcpManager.Logger.Debug("server listening", "ipAddr", tcpManager.ipAddr, "name", tcpManager.name)
	tcpManager.server = listener
	tcpManager.openingConnection = make(map[string]net.Conn)
	tcpManager.handlingConnection = make(map[string]net.Conn)
	tcpManager.trustedConnection = make(map[string]bool)
	return tcpManager
}

// ConnectToPeer connect to the peer
func (g *TCPConnManager) ConnectToPeer(ipAddr string) error {
	if ipAddr == g.ipAddr {
		g.Logger.Error("%s: connect to self", "ipAddr", ipAddr)
		return nil
	}
	g.Logger.Debug("Dial", "destIp", ipAddr, "sourceIp", g.ipAddr)
	conn, err := net.Dial("tcp", ipAddr)
	if err != nil {
		g.Logger.Error("%s: connect to peer %s failed", g.name, ipAddr)
		return err
	}
	g.connRWLock.Lock()
	defer g.connRWLock.Unlock()
	g.openingConnection[ipAddr] = conn
	return nil
}

// ClosePeer Close the connection with the peer
func (g *TCPConnManager) ClosePeer(ipAddr string) {
	g.connRWLock.Lock()
	defer g.connRWLock.Unlock()
	conn, ok := g.openingConnection[ipAddr]
	if !ok {
		return
	}
	_ = conn.Close()
	delete(g.openingConnection, ipAddr)
}

func (g *TCPConnManager) StartServer() {
	for {
		conn, err := g.server.Accept()
		if err != nil {
			g.Logger.Error("accept error", slog.String("error", err.Error()), "ipAddr", g.ipAddr)
			continue
		}
		g.Logger.Debug("%s: accept a connection from %s", g.name, conn.RemoteAddr().String())
		go g.handleConn(conn)
	}
}

func (g *TCPConnManager) GetPeerConnection(addr string) net.Conn {
	g.connRWLock.RLock()
	defer g.connRWLock.RUnlock()
	conn, ok := g.openingConnection[addr]
	if !ok {
		return nil
	}
	return conn
}

func (g *TCPConnManager) GetPeerConnectionSize() int {
	g.connRWLock.RLock()
	defer g.connRWLock.RUnlock()
	return len(g.openingConnection)
}

func (g *TCPConnManager) BroadcastMessageToPeer(frame *model.CommonFrame, degree int) int {
	g.connRWLock.RLock()
	var failedConnection = make([]string, 0)
	for addr, conn := range g.openingConnection {
		if degree == 0 {
			break
		}
		degree--
		err := g.SendMessage(conn, frame)
		if err != nil {
			failedConnection = append(failedConnection, addr)
			continue //this connection is not available
		}
	}
	g.connRWLock.RUnlock()

	for _, addr := range failedConnection {
		g.ClosePeer(addr)
	}

	return degree
}

func (g *TCPConnManager) Stop() {
	g.Logger.Debug("stop the gossip server", "ipAddr", g.ipAddr)
	_ = g.server.Close()
	//close all the connection
	g.connRWLock.Lock()
	for _, conn := range g.openingConnection {
		_ = conn.Close()
	}
	for _, conn := range g.handlingConnection {
		_ = conn.Close()
	}
	g.connRWLock.Unlock()
	g.closeFlag = true
}

func (g *TCPConnManager) isClosed() bool {
	g.closeFlagMutex.Lock()
	defer g.closeFlagMutex.Unlock()
	return g.closeFlag
}

func (g *TCPConnManager) setClosed(flag bool) {
	g.closeFlagMutex.Lock()
	defer g.closeFlagMutex.Unlock()
	g.closeFlag = flag
}

func (g *TCPConnManager) IsTrustedConnection(conn net.Conn) bool {
	g.connRWLock.RLock()
	defer g.connRWLock.RUnlock()
	return g.trustedConnection[conn.RemoteAddr().String()]
}

func (g *TCPConnManager) PutConnection(conn net.Conn) {
	g.connRWLock.Lock()
	defer g.connRWLock.Unlock()
	g.handlingConnection[conn.RemoteAddr().String()] = conn
	g.trustedConnection[conn.RemoteAddr().String()] = false
}

func (g *TCPConnManager) TrustConnection(conn net.Conn) {
	g.connRWLock.Lock()
	defer g.connRWLock.Unlock()
	g.trustedConnection[conn.RemoteAddr().String()] = true
}

func (g *TCPConnManager) RemoveConnection(conn net.Conn) {
	g.connRWLock.Lock()
	defer g.connRWLock.Unlock()
	delete(g.handlingConnection, conn.RemoteAddr().String())
	delete(g.trustedConnection, conn.RemoteAddr().String()) //we don't trust it anymore
}

func (g *TCPConnManager) GetConnectionSize() int {
	g.connRWLock.RLock()
	defer g.connRWLock.RUnlock()
	return len(g.handlingConnection)
}

func (g *TCPConnManager) GetConnectionByAddr(addr string) net.Conn {
	g.connRWLock.RLock()
	defer g.connRWLock.RUnlock()
	conn, ok := g.handlingConnection[addr]
	if !ok {
		return nil
	}
	return conn
}

func (g *TCPConnManager) SendMessage(conn net.Conn, frame *model.CommonFrame) error {
	_, err := conn.Write(frame.Pack())
	if err != nil {
		g.Logger.Error("%s: send message error with conn %s", g.name, conn.RemoteAddr().String())
		return err
	}
	g.Logger.Info("send", "frame", frame.ToString(), "dest", conn.RemoteAddr().String())
	return nil
}

func (g *TCPConnManager) handleConn(conn net.Conn) {
	g.incConnectionCnt()
	//add the connection to the handlingConnection map
	g.PutConnection(conn)
	//will keep this connection util close
	headerBuffer := make([]byte, 4)
	var cnt uint16
	inputFrameChan := make(chan *model.CommonFrame, MaxFrameBuffer)
	defer func() {
		g.decConnectionCnt()
		g.RemoveConnection(conn)
		_ = conn.Close()
		close(inputFrameChan)
	}()
	go g.StartFrameHandler(inputFrameChan, conn) //start handle the frame from this conn
	for {
		cnt = 0
		if g.isClosed() {
			g.Logger.Debug("%s: close the connection with %s", g.name, conn.RemoteAddr().String(), slog.String("error", "server closed"))
			return
		}
		frame := new(model.CommonFrame)
		_, err := conn.Read(headerBuffer)
		if err != nil && err != io.EOF {
			g.Logger.Error("%s: read header error with conn %s", g.name, conn.RemoteAddr().String(), slog.String("error", err.Error()))
			return
		}
		//parse the header
		frame.ParseHeader(headerBuffer)
		//read the payload
		if frame.Size > model.MAX_DATA_SIZE {
			g.Logger.Error("%s: frame size exceed the max size with conn %s", g.name, conn.RemoteAddr().String())
			return
		}
		payloadBuffer := make([]byte, frame.Size)
		for cnt < frame.Size {
			n, err := conn.Read(payloadBuffer[cnt:])
			if err != nil && err != io.EOF {
				g.Logger.Error("%s: read payload error with conn %s", g.name, conn.RemoteAddr().String())
				return
			}
			cnt += uint16(n)
		}
		frame.Payload = payloadBuffer
		//put the frame to the inputFrameChan
		g.Logger.Info("receive", "frame", frame.ToString(), "source", conn.RemoteAddr().String())
		inputFrameChan <- frame
	}
}

func (g *TCPConnManager) StartFrameHandler(inputFrameChan chan *model.CommonFrame, conn net.Conn) {
	defer conn.Close()
	for {
		select {
		case frame := <-inputFrameChan:
			if frame == nil {
				continue
			}
			success, err := g.HandleFrame(frame, conn, g)
			if !success || err != nil {
				g.Logger.Error("%s: handle frame error with conn %s", g.name, conn.RemoteAddr().String())
				return
			}
		}
	}
}
func (g *TCPConnManager) GetConnectionCnt() int {
	g.cntRWLock.RLock()
	defer g.cntRWLock.RUnlock()
	return g.connectionCnt
}

func (g *TCPConnManager) incConnectionCnt() {
	g.cntRWLock.Lock()
	defer g.cntRWLock.Unlock()
	g.connectionCnt++
}

func (g *TCPConnManager) decConnectionCnt() {
	g.cntRWLock.Lock()
	defer g.cntRWLock.Unlock()
	g.connectionCnt--
}
