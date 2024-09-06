package model

import "encoding/binary"

type CommonFrame struct {
	Size    uint16 //size of the message in bytes
	Type    uint16
	Payload []byte
}

func (c *CommonFrame) Clear() {
	c.Size = 0
	c.Type = 0
	c.Payload = nil
}

func (c *CommonFrame) ParseHeader(buffer []byte) {
	c.Size = binary.BigEndian.Uint16(buffer[:2])
	c.Type = binary.BigEndian.Uint16(buffer[2:])
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

func (g *GossipAnnounceMessage) unpack(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	g.TTL = data[0]
	g.Reserved = data[1]
	g.DataType = binary.BigEndian.Uint16(data[2:4])
	g.Data = data[4:]
	return true
}

func (g *GossipNotifyMessage) unpack(data []byte) bool {
	if len(data) != 2 {
		return false
	}
	g.Reserved = data[0]
	g.DataType = data[1]
	return true
}

func (g *GossipNotificationMessage) unpack(data []byte) bool {
	if len(data) < 4 {
		return false
	}
	g.MessageID = binary.BigEndian.Uint16(data[:2])
	g.DataType = binary.BigEndian.Uint16(data[2:4])
	g.Data = data[4:]
	return true
}

func (g *GossipValidationMessage) unpack(data []byte) bool {
	if len(data) != 4 {
		return false
	}
	g.MessageID = binary.BigEndian.Uint16(data[:2])
	g.Validation = binary.BigEndian.Uint16(data[2:4])
	return true
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
	Id [ID_SIZE]byte
}

type PeerValidationMessage struct {
	MessageID  uint16
	Validation uint8
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

func (p *PeerBroadcastMessage) unpack(data []byte) bool {
	if len(data) < 11 {
		return false
	}
	p.Id = binary.BigEndian.Uint64(data[:8])
	p.Ttl = data[8]
	p.Datatype = binary.BigEndian.Uint16(data[9:11])
	p.Data = data[11:]
	return true
}

func (p *PeerRequestMessage) unpack(data []byte) bool {
	if len(data) != ID_SIZE {
		return false
	}
	p.Id = [ID_SIZE]byte{}
	copy(p.Id[:], data[:ID_SIZE])
	return true
}

func (p *PeerDiscoveryMessage) unpack(data []byte) bool {
	if len(data) != 0 {
		return false
	}
	return true
}

func (p *PeerValidationMessage) unpack(data []byte) bool {
	if len(data) != 3 {
		return false
	}
	p.MessageID = binary.BigEndian.Uint16(data[:2])
	p.Validation = data[2]
	return true
}

func (p *PeerInfoMessage) unpack(data []byte) bool {
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
