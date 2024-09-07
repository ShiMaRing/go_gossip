package connection

import (
	"net"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	configPath := "../resources/config_1.ini"
	peerServer, gossipServer := NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_2.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_3.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_4.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_5.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	chn := make(chan int)
	<-chn
}

func TestSingleServer(t *testing.T) {
	configPath := "../resources/config_1.ini"
	peerServer, gossipServer := NewBothServer(configPath)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	_, err := net.Dial("tcp", "127.0.0.1:6011")
	if err != nil {
		t.Fatal(err)
	}

	chn := make(chan int)
	<-chn
}
