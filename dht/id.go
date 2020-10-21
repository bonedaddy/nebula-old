package dht

import (
	"crypto/sha256"

	"github.com/multiformats/go-base36"
	"github.com/slackhq/nebula/cert"
)

// ID wraps a nebula certificate providing dht node id tooling
type ID struct {
	nc *cert.NebulaCertificate
}

// NewID returns a new node id helper
func NewID(nc *cert.NebulaCertificate) *ID { return &ID{nc: nc} }

// PeerID returns the base36 encoded peerID of the nebula peer
func (id *ID) PeerID() (string, error) {
	data, err := id.nc.MarshalToPEM()
	if err != nil {
		return "", err
	}
	hpid := sha256.Sum256(data)
	return base36.EncodeToStringUc(hpid[:]), nil
}
