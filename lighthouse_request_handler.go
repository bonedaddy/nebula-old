package nebula

import (
	"time"

	"github.com/gogo/protobuf/proto"
	"github.com/slackhq/nebula/cert"
	"go.uber.org/zap"
)

type LightHouseHandler struct {
	lh   *LightHouse
	nb   []byte
	out  []byte
	meta *NebulaMeta
	iap  []IpAndPort
	iapp []*IpAndPort
}

func (lh *LightHouse) NewRequestHandler() *LightHouseHandler {
	lhh := &LightHouseHandler{
		lh:  lh,
		nb:  make([]byte, 12, 12),
		out: make([]byte, mtu),

		meta: &NebulaMeta{
			Details: &NebulaMetaDetails{},
		},
	}

	lhh.resizeIpAndPorts(10)

	return lhh
}

// This

// This method is similar to Reset(), but it re-uses the pointer structs
// so that we don't have to re-allocate them
func (lhh *LightHouseHandler) resetMeta() *NebulaMeta {
	details := lhh.meta.Details

	details.Reset()
	lhh.meta.Reset()
	lhh.meta.Details = details

	return lhh.meta
}

func (lhh *LightHouseHandler) resizeIpAndPorts(n int) {
	if cap(lhh.iap) < n {
		lhh.iap = make([]IpAndPort, n)
		lhh.iapp = make([]*IpAndPort, n)

		for i := range lhh.iap {
			lhh.iapp[i] = &lhh.iap[i]
		}
	}
	lhh.iap = lhh.iap[:n]
	lhh.iapp = lhh.iapp[:n]
}

func (lhh *LightHouseHandler) setIpAndPortsFromNetIps(ips []udpAddr) []*IpAndPort {
	lhh.resizeIpAndPorts(len(ips))
	for i, e := range ips {
		lhh.iap[i] = NewIpAndPortFromUDPAddr(e)
	}
	return lhh.iapp
}

func (lhh *LightHouseHandler) HandleRequest(rAddr *udpAddr, vpnIp uint32, p []byte, c *cert.NebulaCertificate, f EncWriter) {
	lh := lhh.lh
	n := lhh.resetMeta()
	err := proto.UnmarshalMerge(p, n)
	if err != nil {
		l.Error(
			"failed to unmarshal lighthouse packet",
			zap.Uint32("vpnIp", uint32(IntIp(vpnIp))),
		)
		//TODO: send recv_error?
		return
	}

	if n.Details == nil {
		l.Error(
			"invalid lighthouse update",
			zap.Uint32("vpnIp", uint32(IntIp(vpnIp))),
		)
		//TODO: send recv_error?
		return
	}

	lh.metricRx(n.Type, 1)

	switch n.Type {
	case NebulaMeta_HostQuery:
		// Exit if we don't answer queries
		if !lh.amLighthouse {
			l.Sugar().Debugf("I don't answer queries, but received from: ", rAddr)

			return
		}

		//l.Debugln("Got Query")
		ips, err := lh.Query(n.Details.VpnIp, f)
		if err != nil {
			//l.Debugf("Can't answer query %s from %s because error: %s", IntIp(n.Details.VpnIp), rAddr, err)
			return
		} else {
			reqVpnIP := n.Details.VpnIp
			n = lhh.resetMeta()
			n.Type = NebulaMeta_HostQueryReply
			n.Details.VpnIp = reqVpnIP
			n.Details.IpAndPorts = lhh.setIpAndPortsFromNetIps(ips)
			reply, err := proto.Marshal(n)
			if err != nil {
				l.Error(
					"failed to marshal lighthouse host query reply",
					zap.Uint32("vpnIp", uint32(IntIp(vpnIp))),
					zap.Error(err),
				)
				return
			}
			lh.metricTx(NebulaMeta_HostQueryReply, 1)
			f.SendMessageToVpnIp(lightHouse, 0, vpnIp, reply, lhh.nb, lhh.out[:0])

			// This signals the other side to punch some zero byte udp packets
			ips, err = lh.Query(vpnIp, f)
			if err != nil {
				l.Debug(
					"cant notify host to punch",
					zap.Uint32("vpnIp", uint32(IntIp(vpnIp))),
				)
				return
			} else {
				//l.Debugln("Notify host to punch", iap)
				n = lhh.resetMeta()
				n.Type = NebulaMeta_HostPunchNotification
				n.Details.VpnIp = vpnIp
				n.Details.IpAndPorts = lhh.setIpAndPortsFromNetIps(ips)
				reply, _ := proto.Marshal(n)
				lh.metricTx(NebulaMeta_HostPunchNotification, 1)
				f.SendMessageToVpnIp(lightHouse, 0, reqVpnIP, reply, lhh.nb, lhh.out[:0])
			}
			//fmt.Println(reply, remoteaddr)
		}

	case NebulaMeta_HostQueryReply:
		if !lh.IsLighthouseIP(vpnIp) {
			return
		}
		for _, a := range n.Details.IpAndPorts {
			//first := n.Details.IpAndPorts[0]
			ans := NewUDPAddr(a.Ip, uint16(a.Port))
			lh.AddRemote(n.Details.VpnIp, ans, false)
		}
		// Non-blocking attempt to trigger, skip if it would block
		select {
		case lh.handshakeTrigger <- n.Details.VpnIp:
		default:
		}

	case NebulaMeta_HostUpdateNotification:
		//Simple check that the host sent this not someone else
		if n.Details.VpnIp != vpnIp {
			l.Debug(
				"host sent invalid update",
				zap.Uint32("vpnIp", uint32(IntIp(vpnIp))),
				zap.Uint32("answer", uint32(IntIp(n.Details.VpnIp))),
			)
			return
		}
		for _, a := range n.Details.IpAndPorts {
			ans := NewUDPAddr(a.Ip, uint16(a.Port))
			lh.AddRemote(n.Details.VpnIp, ans, false)
		}
	case NebulaMeta_HostMovedNotification:
	case NebulaMeta_HostPunchNotification:
		if !lh.IsLighthouseIP(vpnIp) {
			return
		}

		empty := []byte{0}
		for _, a := range n.Details.IpAndPorts {
			vpnPeer := NewUDPAddr(a.Ip, uint16(a.Port))
			go func() {
				time.Sleep(lh.punchDelay)
				lh.metricHolepunchTx.Inc(1)
				lh.punchConn.WriteTo(empty, vpnPeer)

			}()
			l.Sugar().Debugf("Punching %s on %d for %s", IntIp(a.Ip), a.Port, IntIp(n.Details.VpnIp))
		}
		// This sends a nebula test packet to the host trying to contact us. In the case
		// of a double nat or other difficult scenario, this may help establish
		// a tunnel.
		if lh.punchBack {
			go func() {
				time.Sleep(time.Second * 5)
				l.Sugar().Debugf("Sending a nebula test packet to vpn ip %s", IntIp(n.Details.VpnIp))
				// TODO we have to allocate a new output buffer here since we are spawning a new goroutine
				// for each punchBack packet. We should move this into a timerwheel or a single goroutine
				// managed by a channel.
				f.SendMessageToVpnIp(test, testRequest, n.Details.VpnIp, []byte(""), make([]byte, 12), make([]byte, mtu))
			}()
		}
	}
}
