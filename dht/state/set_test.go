package state_test

import (
	"bytes"
	"crypto/sha256"
	"log"
	"net"
	"testing"
	"time"

	"github.com/multiformats/go-base36"
	"github.com/slackhq/nebula/cert"
	"github.com/slackhq/nebula/dht/state"
)

func TestSet_Insert(t *testing.T) {
	s := state.NewSet(10)

	id := ID()

	s = s.Insert(id)

	if s.IndexOf(id) != 0 {
		t.Error("failed to insert id")
	}
}

func TestSet_Remove(t *testing.T) {
	s := state.NewSet(10)

	id := ID()

	s = s.Insert(id)
	if s.IndexOf(id) != 0 {
		t.Error("failed to insert id")
	}

	s, ok := s.Remove(id)
	if !ok {
		t.Error("failed to remove")
	}

	if s.IndexOf(id) != -1 {
		t.Error("failed to remove id")
	}
}

func TestSet_Closest(t *testing.T) {
	s := state.NewSet(10)

	first := ID()

	search := UpperID(first)

	second := UpperID(search)

	s = s.Insert(first)
	s = s.Insert(second)

	if !bytes.Equal(first, s.Closest(search)) {
		t.Error("unexpected closest value")
	}
}

func TestSet_Insert_IsProperlySorted(t *testing.T) {
	s := state.NewSet(10)

	first := ID()
	second := UpperID(first)
	last := UpperID(second)

	s = s.Insert(first)
	s = s.Insert(second)
	s = s.Insert(last)

	if s.IndexOf(first) != 2 {
		t.Fatal("incorrect sorting")
	}

	if s.IndexOf(second) != 1 {
		t.Fatal("incorrect sorting")
	}

	if s.IndexOf(last) != 0 {
		t.Fatal("incorrect sorting")
	}
}

func TestSet_Insert_IsProperlySorted_Reverse(t *testing.T) {
	s := state.NewSet(10)

	first := ID()
	second := LowerID(first)
	last := LowerID(second)

	s = s.Insert(first)
	s = s.Insert(second)
	s = s.Insert(last)

	if s.IndexOf(first) != 0 {
		t.Fatal("incorrect sorting")
	}

	if s.IndexOf(second) != 1 {
		t.Fatal("incorrect sorting")
	}

	if s.IndexOf(last) != 2 {
		t.Fatal("incorrect sorting")
	}
}

func TestSet_Insert_Max_Length(t *testing.T) {
	prev := ID()

	length := 10
	s := state.NewSet(length)

	for i := 0; i < length; i++ {
		s = s.Insert(prev)
		prev = UpperID(prev)
	}

	next := UpperID(prev)
	s = s.Insert(next)

	if s.Length() > length {
		t.Fatal("list too long")
	}

	if !bytes.Equal(s.Get(0), next) {
		t.Fatal("unexpected value")
	}
}

func UpperID(id state.Peer) state.Peer {
	n := make(state.Peer, len(id))
	copy(n[:], id[:])

	i := 2

	for ; i <= len(id); i++ {
		if id[i] < 255 {
			break
		}
	}

	n[i] += 1
	return n
}

func LowerID(id state.Peer) state.Peer {
	n := make(state.Peer, len(id))
	copy(n[:], id[:])

	i := 2

	for ; i <= len(id); i++ {
		if id[i] > 0 {
			break
		}
	}

	n[i] -= 1

	return n
}

func ID() []byte {
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

	data, err := nc.MarshalToPEM()
	if err != nil {
		log.Fatal(err)
	}
	hpid := sha256.Sum256(data)
	return []byte(base36.EncodeToStringUc(hpid[:]))
}
