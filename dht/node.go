package dht

import (
	"context"
	"log"

	"github.com/libp2p/go-eventbus"
	"github.com/libp2p/go-libp2p-core/event"
	peer "github.com/libp2p/go-libp2p-peer"
	"github.com/slackhq/nebula/dht/pb"
	"github.com/slackhq/nebula/dht/state"
)

type Node struct {
	*DHT
	sub event.Subscription
	ctx context.Context
}

func NewNode(ctx context.Context, dht *DHT) (*Node, error) {
	sub, err := newSubscription()
	if err != nil {
		return nil, err
	}
	return &Node{DHT: dht, sub: sub, ctx: ctx}, nil
}

func newSubscription() (event.Subscription, error) {
	evBus := eventbus.NewBus()
	evts := []interface{}{
		// register for event bus notifications of when peers successfully complete identification in order to update
		// the routing table
		new(event.EvtPeerIdentificationCompleted),

		// register for event bus protocol ID changes in order to update the routing table
		new(event.EvtPeerProtocolsUpdated),

		// register for event bus notifications for when our local address/addresses change so we can
		// advertise those to the network
		new(event.EvtLocalAddressesUpdated),
	}
	return evBus.Subscribe(evts, eventbus.BufSize(256))
}

func (n *Node) poll(s event.Subscription) {
	defer s.Close()

	for {
		select {
		case e, more := <-n.sub.Out():
			if !more {
				return
			}

			switch evt := e.(type) {
			case event.EvtLocalAddressesUpdated:
				// when our address changes, we should proactively tell our closest peers about it so
				// we become discoverable quickly. The Identify protocol will push a signed peer record
				// with our new address to all peers we are connected to. However, we might not necessarily be connected
				// to our closet peers & so in the true spirit of Zen, searching for ourself in the network really is the best way
				// to to forge connections with those matter.
				n.handleUpdate()
			case event.EvtPeerProtocolsUpdated:
				n.handlePeerChangeEvent(evt.Peer)
			case event.EvtPeerIdentificationCompleted:
				n.handlePeerChangeEvent(evt.Peer)
			default:
				// something has gone really wrong if we get an event for another type
				log.Printf("got wrong type from subscription: %T", e)
			}
		case <-n.ctx.Done():
			return
		}
	}
}

func (n *Node) handlePeerChangeEvent(p peer.ID) {
	n.AddPeer([]byte(p))

	// @todo
	n.MapNeighbors(func(peer state.Peer) {
		go func() {
			// @todo according to pastry this `Key` should be different than the peer we are sending to
			err := n.Send(
				n.ctx,
				&pb.Message{Key: peer, Type: pb.Message_NODE_ANNOUNCE, Sender: []byte(p)},
			)

			if err != nil {
				log.Println(err)
			}
		}()
	})
}

func (n *Node) handleUpdate() {
	n.MapNeighbors(func(peer state.Peer) {
		go func() {
			// @todo according to pastry this `Key` should be different than the peer we are sending to
			err := n.Send(
				n.ctx,
				&pb.Message{Key: peer, Type: pb.Message_NODE_JOIN, Sender: n.ID},
			)

			if err != nil {
				log.Println(err)
			}
		}()
	})
}
