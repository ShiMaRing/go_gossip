package model

const (
	MAX_DATA_SIZE = 4096
	ID_SIZE       = 16
	FRAME_BUFFER  = 16

	PEER_BROADCAST_MESSAGE  = 510
	PEER_VALIDATION_MESSAGE = 511
	PEER_DISCOVERY_MESSAGE  = 512
	PEER_INFO_MESSAGE       = 513
	PEER_REQUEST_MESSAGE    = 514

	GOSSIP_ANNOUCE_MESSAGE = 500
	GOSSIP_NOTIFY          = 501
	GOSSIP_NOTIFICATION    = 502
	GOSSIP_VALIDATION      = 503
)
