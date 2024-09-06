package connection

import (
	"go_gossip/config"
	"go_gossip/model"
	"go_gossip/utils"
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

	openingConnection map[string]net.Conn //all connection that we connect to
	connRWLock        sync.RWMutex        //lock for the handlingConnection
	cntRWLock         sync.RWMutex
	closeFlagMutex    sync.Mutex

	maxConnCnt int
	minConnCnt int
}

// NewTCPManager Base on the config apiAddress to create a gossip server
func NewTCPManager(name string, ipAddr string) *TCPConnManager {
	var tcpManager = &TCPConnManager{}
	tcpManager.ipAddr = ipAddr
	tcpManager.name = name
	tcpManager.minConnCnt = config.P2PConfig.MinConnections
	tcpManager.maxConnCnt = config.P2PConfig.MaxConnections
	listener, err := net.Listen("tcp", tcpManager.ipAddr)
	if err != nil { // when we start gossip server fail,just panic
		utils.Logger.Error("%s: api server listen failed", tcpManager.name, err)
		panic(err)
	}
	defer listener.Close()
	//now the gossip server is listening
	tcpManager.server = listener
	return tcpManager
}

func (g *TCPConnManager) Start() {
	for {
		conn, err := g.server.Accept()
		if err != nil {
			utils.Logger.Error("accept error", err)
			break
		}
		utils.Logger.Debug("%s: accept a connection from %s", g.name, conn.RemoteAddr().String())
		go g.handleConn(conn)
	}
}

func (g *TCPConnManager) Stop() {
	_ = g.server.Close()
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
			utils.Logger.Debug("%s: close the connection with %s", g.name, conn.RemoteAddr().String())
			return
		}
		frame := new(model.CommonFrame)
		_, err := conn.Read(headerBuffer)
		if err != nil {
			utils.Logger.Error("%s: read header error with conn %s", g.name, conn.RemoteAddr().String())
			return
		}
		//parse the header
		frame.ParseHeader(headerBuffer)
		//read the payload
		if frame.Size > model.MAX_DATA_SIZE {
			utils.Logger.Error("%s: frame size exceed the max size with conn %s", g.name, conn.RemoteAddr().String())
			return
		}
		payloadBuffer := make([]byte, frame.Size)
		for cnt < frame.Size {
			n, err := conn.Read(payloadBuffer[cnt:])
			if err != nil {
				utils.Logger.Error("%s: read payload error with conn %s", g.name, conn.RemoteAddr().String())
				return
			}
			cnt += uint16(n)
		}
		frame.Payload = payloadBuffer
		//put the frame to the inputFrameChan
		utils.Logger.Debug("%s receive a frame from %s", g.name, conn.RemoteAddr().String())
		inputFrameChan <- frame
	}
}

func (g *TCPConnManager) StartFrameHandler(inputFrameChan chan *model.CommonFrame, conn net.Conn) {
	defer conn.Close()
	for {
		select {
		case frame := <-inputFrameChan:
			success, err := g.HandleFrame(frame, conn, g)
			if !success || err != nil {
				utils.Logger.Error("%s: handle frame error with conn %s", g.name, conn.RemoteAddr().String())
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
