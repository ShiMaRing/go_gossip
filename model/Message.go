package model

//we will encode and decode the message with json

type CommonFrame struct {
	Length uint16 //size of the message in bytes
	Data   []byte //json data
}

// =========================================================
// Gossip message
// =========================================================

type GossipAnnounceMessage struct {
	Size     uint16 //size of the message in bytes
	Type     uint16
	TTL      uint8
	Reserved uint8
	DataType uint16
	Data     []byte
}

type GossipNotifyMessage struct {
	Size     uint16 //size of the message in bytes
	Type     uint16
	Reserved uint8
	DataType uint8
}

type GossipNotificationMessage struct {
	Size      uint16 //size of the message in bytes
	Type      uint16
	MessageID uint16
	DataType  uint16
	Data      []byte
}

type GossipValidationMessage struct {
	Size       uint16 //size of the message in bytes
	Type       uint16
	MessageID  uint16
	Validation uint16 //only the lowst bit is used
}

// =========================================================
// Peer to Peer message
// =========================================================

type PeerBroadcastMessage struct {
	Size     uint16 //size of the message in bytes
	Type     uint16
	Id       uint64
	Ttl      uint8
	Datatype uint16
	Data     []byte
}

type PeerRequestMessage struct {
	Size uint16 //size of the message in bytes
	Type uint16
	Id   [ID_SIZE]byte
}

type PeerValidationMessage struct {
	Size       uint16 //size of the message in bytes
	Type       uint16
	MessageID  uint16
	Validation uint8
}

type PeerDiscoveryMessage struct {
	Size uint16 //size of the message in bytes
	Type uint16
}

type PeerInfoMessage struct {
	Size  uint16 //size of the message in bytes
	Type  uint16
	Cnt   uint16
	Peers []PeerInfo
}

type PeerInfo struct {
	Size    uint16 //size of the message in bytes
	IP      uint32 // IP of the peer
	P2PPort uint16 // Port of the peer
	APIPort uint16 // Port of the peer
}
