package dht

import (
	"net"
	"testing"
	"time"

	"github.com/slackhq/nebula/cert"
	"github.com/stretchr/testify/assert"
)

func TestID(t *testing.T) {
	before := time.Now().Add(time.Second * -60).Round(time.Second)
	after := time.Now().Add(time.Second * 60).Round(time.Second)
	pubKey := []byte("1234567890abcedfghij1234567890ab")

	nc := cert.NebulaCertificate{
		Details: cert.NebulaCertificateDetails{
			Name: "testing",
			Ips: []*net.IPNet{
				{IP: net.ParseIP("10.1.1.1"), Mask: net.IPMask(net.ParseIP("255.255.255.0"))},
				{IP: net.ParseIP("10.1.1.2"), Mask: net.IPMask(net.ParseIP("255.255.0.0"))},
				{IP: net.ParseIP("10.1.1.3"), Mask: net.IPMask(net.ParseIP("255.0.255.0"))},
			},
			Subnets: []*net.IPNet{
				{IP: net.ParseIP("9.1.1.1"), Mask: net.IPMask(net.ParseIP("255.0.255.0"))},
				{IP: net.ParseIP("9.1.1.2"), Mask: net.IPMask(net.ParseIP("255.255.255.0"))},
				{IP: net.ParseIP("9.1.1.3"), Mask: net.IPMask(net.ParseIP("255.255.0.0"))},
			},
			Groups:    []string{"test-group1", "test-group2", "test-group3"},
			NotBefore: before,
			NotAfter:  after,
			PublicKey: pubKey,
			IsCA:      false,
			Issuer:    "1234567890abcedfghij1234567890ab",
		},
	}

	id := NewID(&nc)

	pid, err := id.PeerID()
	assert.NoError(t, err)
	t.Logf("peer id: %s", pid)
}
