package core

import (
	"github.com/libp2p/go-libp2p/core/peer"
)

type Payload struct {
	Hash         string  `json:"hash"`
	PeerIdentity peer.ID `json:"identity"`
}
