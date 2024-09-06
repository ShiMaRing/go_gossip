package model

import "encoding/binary"

//we will encode and decode the message with json

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
	P2PPort uint16 // Port of the peer
	APIPort uint16 // Port of the peer
}
