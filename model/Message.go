package model

import (
	"encoding/binary"
	"fmt"
)

type CommonFrame struct {
	Size    uint16 //size of the message in bytes
	Type    uint16
	Payload []byte
}

func (f *CommonFrame) Clear() {
	f.Size = 0
	f.Type = 0
	f.Payload = nil
}

func (f *CommonFrame) ParseHeader(buffer []byte) {
	f.Size = binary.BigEndian.Uint16(buffer[:2])
	f.Type = binary.BigEndian.Uint16(buffer[2:])
}

// =========================================================
// Gossip message
// =========================================================

type GossipAnnounceMessage struct {
	TTL      uint8
	Reserved uint8
	DataType uint16
	Data     []byte
}

type GossipNotifyMessage struct {
	Reserved uint8
	DataType uint8
}

type GossipNotificationMessage struct {
	MessageID uint16
	DataType  uint16
	Data      []byte
}

type GossipValidationMessage struct {
	MessageID  uint16
	Validation uint16 //only the lowst bit is used
}

func (f *CommonFrame) Pack() []byte {
	data := make([]byte, 4+len(f.Payload))
	binary.BigEndian.PutUint16(data[:2], f.Size)
	binary.BigEndian.PutUint16(data[2:4], f.Type)
	copy(data[4:], f.Payload)
	return data
}

func (f *CommonFrame) ToString() string {
	var frameType string
	switch f.Type {
	case GOSSIP_NOTIFY:
		frameType = "GOSSIP_NOTIFY"
	case GOSSIP_ANNOUCE:
		frameType = "GOSSIP_ANNOUNCE"
	case GOSSIP_NOTIFICATION:
		frameType = "GOSSIP_NOTIFICATION"
	case GOSSIP_VALIDATION:
		frameType = "GOSSIP_VALIDATION"
	case PEER_BROADCAST:
		frameType = "PEER_BROADCAST"
	case PEER_REQUEST:
		frameType = "PEER_REQUEST"
	case PEER_VALIDATION:
		frameType = "PEER_VALIDATION"
	case PEER_DISCOVERY:
		frameType = "PEER_DISCOVERY"
	case PEER_INFO:
		frameType = "PEER_INFO"
	default:
		frameType = "UNKNOWN"
	}
	return fmt.Sprintf("Size: %d, Type: %s", f.Size, frameType)
}

func (g *GossipAnnounceMessage) Pack() []byte {
	data := make([]byte, 4+len(g.Data))
	data[0] = g.TTL
	data[1] = g.Reserved
	binary.BigEndian.PutUint16(data[2:4], g.DataType)
	copy(data[4:], g.Data)
	return data
}

func (g *GossipAnnounceMessage) Unpack(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	g.TTL = data[0]
	g.Reserved = data[1]
	g.DataType = binary.BigEndian.Uint16(data[2:4])
	g.Data = data[4:]
	return true
}

func (g *GossipAnnounceMessage) ToString() string {
	return fmt.Sprintf("TTL: %d, DataType: %d, Data: %s", g.TTL, g.DataType, string(g.Data))
}

func (g *GossipNotifyMessage) Pack() []byte {
	data := make([]byte, 2)
	data[0] = g.Reserved
	data[1] = g.DataType
	return data
}

func (g *GossipNotifyMessage) Unpack(data []byte) bool {
	if len(data) != 2 {
		return false
	}
	g.Reserved = data[0]
	g.DataType = data[1]
	return true
}

func (g *GossipNotificationMessage) Pack() []byte {
	data := make([]byte, 4+len(g.Data))
	binary.BigEndian.PutUint16(data[:2], g.MessageID)
	binary.BigEndian.PutUint16(data[2:4], g.DataType)
	copy(data[4:], g.Data)
	return data
}

func (g *GossipNotificationMessage) Unpack(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	g.MessageID = binary.BigEndian.Uint16(data[:2])
	g.DataType = binary.BigEndian.Uint16(data[2:4])
	g.Data = data[4:]
	return true
}

func (g *GossipNotificationMessage) ToString() string {
	return fmt.Sprintf("MessageID: %d, DataType: %d, Data: %s", g.MessageID, g.DataType, string(g.Data))
}

func (g *GossipValidationMessage) Pack() []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[:2], g.MessageID)
	binary.BigEndian.PutUint16(data[2:4], g.Validation)
	return data
}

func (g *GossipValidationMessage) Unpack(data []byte) bool {
	if len(data) != 4 {
		return false
	}
	g.MessageID = binary.BigEndian.Uint16(data[:2])
	g.Validation = binary.BigEndian.Uint16(data[2:4])
	return true
}

func (g *GossipValidationMessage) ToString() string {
	return fmt.Sprintf("MessageID: %d, Validation: %d", g.MessageID, g.Validation)
}

// =========================================================
// Peer to Peer message
// =========================================================

type PeerBroadcastMessage struct {
	Id       uint64
	Ttl      uint8
	Datatype uint16
	Data     []byte
}

type PeerRequestMessage struct {
	MessageID uint16
	PeerInfo  PeerInfo //request with my self info
}

type PeerValidationMessage struct {
	MessageID  uint16
	Validation uint16
}

type PeerDiscoveryMessage struct {
}

type PeerInfoMessage struct {
	Cnt   uint16
	Peers []PeerInfo
}

type PeerInfo struct {
	P2PIP   uint32
	P2PPort uint16 // Port of the peer
	ApiIP   uint32
	APIPort uint16 // Port of the peer
}

func (p *PeerBroadcastMessage) Pack() []byte {
	data := make([]byte, 11+len(p.Data))
	binary.BigEndian.PutUint64(data[:8], p.Id)
	data[8] = p.Ttl
	binary.BigEndian.PutUint16(data[9:11], p.Datatype)
	copy(data[11:], p.Data)
	return data
}

func (p *PeerBroadcastMessage) Unpack(data []byte) bool {
	if len(data) < 11 {
		return false
	}
	p.Id = binary.BigEndian.Uint64(data[:8])
	p.Ttl = data[8]
	p.Datatype = binary.BigEndian.Uint16(data[9:11])
	p.Data = data[11:]
	return true
}

func (p *PeerRequestMessage) Pack() []byte {
	data := make([]byte, 2+12) //the peer info size is 4+2+4+2
	binary.BigEndian.PutUint16(data[:2], p.MessageID)
	binary.BigEndian.PutUint32(data[2:6], p.PeerInfo.P2PIP)
	binary.BigEndian.PutUint16(data[6:8], p.PeerInfo.P2PPort)
	binary.BigEndian.PutUint32(data[8:12], p.PeerInfo.ApiIP)
	binary.BigEndian.PutUint16(data[12:14], p.PeerInfo.APIPort)
	return data
}

func (p *PeerRequestMessage) Unpack(data []byte) bool {
	if len(data) != 14 {
		return false
	}
	p.MessageID = binary.BigEndian.Uint16(data[:2])
	p.PeerInfo.P2PIP = binary.BigEndian.Uint32(data[2:6])
	p.PeerInfo.P2PPort = binary.BigEndian.Uint16(data[6:8])
	p.PeerInfo.ApiIP = binary.BigEndian.Uint32(data[8:12])
	p.PeerInfo.APIPort = binary.BigEndian.Uint16(data[12:14])
	return true
}

func (p *PeerDiscoveryMessage) Pack() []byte {
	return []byte{}
}

func (p *PeerDiscoveryMessage) Unpack(data []byte) bool {
	if len(data) != 0 {
		return false
	}
	return true
}

func (p *PeerValidationMessage) Pack() []byte {
	data := make([]byte, 4)
	binary.BigEndian.PutUint16(data[:2], p.MessageID)
	binary.BigEndian.PutUint16(data[2:4], p.Validation)
	return data
}

func (p *PeerValidationMessage) Unpack(data []byte) bool {
	if len(data) != 4 {
		return false
	}
	p.MessageID = binary.BigEndian.Uint16(data[:2])
	p.Validation = binary.BigEndian.Uint16(data[2:4])
	return true
}

func (p *PeerInfoMessage) Pack() []byte {
	data := make([]byte, 2+int(p.Cnt)*12)
	prev := data
	binary.BigEndian.PutUint16(data[:2], p.Cnt)
	data = data[2:]
	for i := 0; i < int(p.Cnt); i++ {
		binary.BigEndian.PutUint32(data[:4], p.Peers[i].P2PIP)
		binary.BigEndian.PutUint16(data[4:6], p.Peers[i].P2PPort)
		binary.BigEndian.PutUint32(data[6:10], p.Peers[i].ApiIP)
		binary.BigEndian.PutUint16(data[10:12], p.Peers[i].APIPort)
		data = data[12:]
	}
	return prev
}

func (p *PeerInfoMessage) Unpack(data []byte) bool {
	if len(data) < 2 {
		return false
	}
	p.Cnt = binary.BigEndian.Uint16(data[:2])
	if len(data) != 2+int(p.Cnt)*12 { //the size of the frame is not correct
		return false
	}
	data = data[2:]
	p.Peers = make([]PeerInfo, p.Cnt)
	for i := 0; i < int(p.Cnt); i++ {
		p.Peers[i].P2PIP = binary.BigEndian.Uint32(data[:4])
		p.Peers[i].P2PPort = binary.BigEndian.Uint16(data[4:6])
		p.Peers[i].ApiIP = binary.BigEndian.Uint32(data[6:10])
		p.Peers[i].APIPort = binary.BigEndian.Uint16(data[10:12])
		data = data[12:]
	}
	return true
}

func MakeCommonFrame(messageType uint16, data []byte) *CommonFrame {
	frame := &CommonFrame{}
	frame.Type = messageType
	frame.Payload = make([]byte, len(data))
	copy(frame.Payload, data)
	frame.Size = uint16(len(data))
	return frame
}
