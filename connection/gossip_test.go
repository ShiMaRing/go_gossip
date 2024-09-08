package connection

import (
	"go_gossip/model"
	"io"
	"log/slog"
	"math/rand"
	"net"
	"strconv"
	"testing"
	"time"
)

const (
	TestDataType1 = 1
	TestDataType2 = 2
	TestDataType3 = 3
)

const SubscriberCount = 15

func TestGossip(t *testing.T) {

	var apiServerAddrs []string = make([]string, 0)

	configPath := "../resources/config_1.ini"
	peerServer, gossipServer := NewBothServer(configPath)
	peerServer.PeerServerStart()
	apiServerAddrs = append(apiServerAddrs, peerServer.P2PConfig.APIAddress)
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_2.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	apiServerAddrs = append(apiServerAddrs, peerServer.P2PConfig.APIAddress)
	peerServer.PeerServerStart()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_3.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	apiServerAddrs = append(apiServerAddrs, peerServer.P2PConfig.APIAddress)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_4.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	apiServerAddrs = append(apiServerAddrs, peerServer.P2PConfig.APIAddress)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 2)
	configPath = "../resources/config_5.ini"
	peerServer, gossipServer = NewBothServer(configPath)
	apiServerAddrs = append(apiServerAddrs, peerServer.P2PConfig.APIAddress)
	peerServer.PeerServerStart()
	go gossipServer.Start()

	time.Sleep(time.Second * 5)
	//randomly generate some subscriber

	for i := 0; i < SubscriberCount; i++ {
		randAddr := apiServerAddrs[rand.Intn(len(apiServerAddrs))]
		randDataType := []uint8{TestDataType1, TestDataType2}[rand.Intn(2)]
		idx := strconv.Itoa(i)
		go GenerateSubscriber("subscriber"+idx, randAddr, randDataType)
	}

	rand.Shuffle(len(apiServerAddrs), func(i, j int) {
		apiServerAddrs[i], apiServerAddrs[j] = apiServerAddrs[j], apiServerAddrs[i]
	})

	//generate some api subscriber
	time.Sleep(time.Second * 2)
	AnnounceToAll(apiServerAddrs[0], TestDataType1, []byte("test1"), 3)
	time.Sleep(time.Second * 2)
	AnnounceToAll(apiServerAddrs[1], TestDataType2, []byte("test2"), 3)
	time.Sleep(time.Second * 2)
	AnnounceToAll(apiServerAddrs[2], TestDataType3, []byte("test3"), 3)

	chn := make(chan int)
	<-chn
}

func AnnounceToAll(apiAddr string, dataType uint16, data []byte, ttl uint16) {
	conn, err := net.Dial("tcp", apiAddr)
	if err != nil {
		slog.Default().Error("connect to api server failed", "err", err)
		panic(err)
	}
	defer conn.Close()
	//send a message to announce
	message := model.GossipAnnounceMessage{
		DataType: dataType,
		Data:     data,
		Reserved: 0,
		TTL:      uint8(ttl),
	}
	frame := model.MakeCommonFrame(model.GOSSIP_ANNOUCE, message.Pack())
	//send the frame
	_, err = conn.Write(frame.Pack())
	if err != nil {
		slog.Default().Error("send message failed", "err", err)
		return
	}
	slog.Default().Info("send announce message", "message", message.ToString(), "dest", conn.RemoteAddr().String())
}

func GenerateSubscriber(name string, apiAddr string, dataType uint8) {
	conn, err := net.Dial("tcp", apiAddr)
	if err != nil {
		slog.Default().Error("connect to api server failed", "err", err, "subscriber", name)
		panic(err)
	}
	defer conn.Close()
	slog.Default().Info("connect to api server", "subscriber", name, "apiAddr", apiAddr, "subscribeType", dataType)
	//send a message to subscribe
	message := model.GossipNotifyMessage{
		DataType: uint8(dataType),
		Reserved: 0,
	}
	frame := model.MakeCommonFrame(model.GOSSIP_NOTIFY, message.Pack())
	//send the frame and wait for notification
	_, err = conn.Write(frame.Pack())
	if err != nil {
		slog.Default().Error("send message failed", "err", err, "subscriber", name)
		return
	}
	// read here
	var cnt uint16
	for {
		cnt = 0
		headerBuffer := make([]byte, 4)
		frame := new(model.CommonFrame)
		_, err := conn.Read(headerBuffer)
		if err != nil && err != io.EOF {
			slog.Default().Error("read header error", "err", err, "subscriber", name)
			return
		}
		//parse the header
		frame.ParseHeader(headerBuffer)
		payloadBuffer := make([]byte, frame.Size)
		for cnt < frame.Size {
			n, err := conn.Read(payloadBuffer[cnt:])
			if err != nil && err != io.EOF {
				slog.Default().Error("read payload error", "err", err, "subscriber", name)
				return
			}
			cnt += uint16(n)
		}
		frame.Payload = payloadBuffer
		//it must be a notification
		if frame.Type == model.GOSSIP_NOTIFICATION {
			//get the notification message
			notification := model.GossipNotificationMessage{}
			notification.Unpack(frame.Payload)
			slog.Default().Info("receive a notification", "notification", notification.ToString(), "subscriber", name)

			//then return a validation message
			validation := model.GossipValidationMessage{
				MessageID:  notification.MessageID,
				Validation: 1,
			}
			frame = model.MakeCommonFrame(model.GOSSIP_VALIDATION, validation.Pack())
			var data = frame.Pack()
			cnt = 0
			n, err := conn.Write(data)
			for cnt < uint16(len(data)) {
				n, err = conn.Write(data[cnt:])
				if err != nil {
					slog.Default().Error("send validation message failed", "err", err, "subscriber", name)
					return
				}
				cnt += uint16(n)
			}

			if err != nil {
				slog.Default().Error("send validation message failed", "err", err, "subscriber", name)
				return
			}
			//slog.Default().Info("send validation message", "validation", validation.ToString(), "subscriber", name)
		} else {
			slog.Default().Error("receive a wrong frame", "frame", frame.ToString(), "subscriber", name)
			return
		}

	}

}
