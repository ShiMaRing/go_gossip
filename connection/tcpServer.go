package connection

import (
	"go_gossip/model"
	"go_gossip/utils"
	"net"
	"sync"
)

const MaxFrameBuffer = 16

// TCPServer Gossip The gossip protocol implementation
type TCPServer struct {
	Name          string
	IpAddr        string
	HandleFrame   func(*model.CommonFrame) bool
	CheckFrame    func(*model.CommonFrame) bool
	server        net.Listener
	closeFlag     bool
	connectionCnt int //current connection count

	cntRWLock      sync.RWMutex
	closeFlagMutex sync.Mutex
}

// NewTCPServer Base on the config apiAddress to create a gossip server
func NewTCPServer(name string, ipAddr string) *TCPServer {
	var tcpServer = &TCPServer{}
	tcpServer.IpAddr = ipAddr
	tcpServer.Name = name
	listener, err := net.Listen("tcp", tcpServer.IpAddr)
	if err != nil { // when we start gossip server fail,just panic
		utils.Logger.Error("%s: api server listen failed", tcpServer.Name, err)
		panic(err)
	}
	defer listener.Close()
	//now the gossip server is listening
	tcpServer.server = listener
	return tcpServer
}

func (g *TCPServer) Start() {
	for {
		conn, err := g.server.Accept()
		if err != nil {
			utils.Logger.Error("accept error", err)
			break
		}
		utils.Logger.Debug("%s: accept a connection from %s", g.Name, conn.RemoteAddr().String())
		go g.handleConn(conn)
	}
}

func (g *TCPServer) Stop() {
	_ = g.server.Close()
	g.closeFlag = true
}

func (g *TCPServer) isClosed() bool {
	g.closeFlagMutex.Lock()
	defer g.closeFlagMutex.Unlock()
	return g.closeFlag
}

func (g *TCPServer) setClosed(flag bool) {
	g.closeFlagMutex.Lock()
	defer g.closeFlagMutex.Unlock()
	g.closeFlag = flag
}

func (g *TCPServer) handleConn(conn net.Conn) {
	g.incConnectionCnt()
	//will keep this connection util close
	headerBuffer := make([]byte, 4)
	var cnt uint16
	inputFrameChan := make(chan *model.CommonFrame, MaxFrameBuffer)
	defer func() {
		g.decConnectionCnt()
		_ = conn.Close()
		close(inputFrameChan)
	}()

	go g.StartFrameHandler(inputFrameChan, conn) //start handle the frame from this conn

	for {
		cnt = 0
		if g.isClosed() {
			utils.Logger.Debug("%s: close the connection with %s", g.Name, conn.RemoteAddr().String())
			return
		}
		frame := new(model.CommonFrame)
		_, err := conn.Read(headerBuffer)
		if err != nil {
			utils.Logger.Error("%s: read header error with conn %s", g.Name, conn.RemoteAddr().String())
			return
		}
		//parse the header
		frame.ParseHeader(headerBuffer)
		//read the payload
		if frame.Size > model.MAX_DATA_SIZE {
			utils.Logger.Error("%s: frame size exceed the max size with conn %s", g.Name, conn.RemoteAddr().String())
			return
		}
		payloadBuffer := make([]byte, frame.Size)
		for cnt < frame.Size {
			n, err := conn.Read(payloadBuffer[cnt:])
			if err != nil {
				utils.Logger.Error("%s: read payload error with conn %s", g.Name, conn.RemoteAddr().String())
				return
			}
			cnt += uint16(n)
		}
		frame.Payload = payloadBuffer
		//put the frame to the inputFrameChan
		utils.Logger.Debug("%s receive a frame from %s", g.Name, conn.RemoteAddr().String())
		valid := g.CheckFrame(frame) //discard the invalid frame and close the connection
		if !valid {
			return
		}
		inputFrameChan <- frame
	}
}

func (g *TCPServer) StartFrameHandler(inputFrameChan chan *model.CommonFrame, conn net.Conn) {
	defer conn.Close()
	for {
		select {
		case frame := <-inputFrameChan:
			if !g.HandleFrame(frame) {
				return
			}
		}
	}
}
func (g *TCPServer) GetConnectionCnt() int {
	g.cntRWLock.RLock()
	defer g.cntRWLock.RUnlock()
	return g.connectionCnt
}

func (g *TCPServer) incConnectionCnt() {
	g.cntRWLock.Lock()
	defer g.cntRWLock.Unlock()
	g.connectionCnt++
}

func (g *TCPServer) decConnectionCnt() {
	g.cntRWLock.Lock()
	defer g.cntRWLock.Unlock()
	g.connectionCnt--
}
