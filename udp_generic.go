// +build !linux android

// udp_generic implements the nebula UDP interface in pure Go stdlib. This
// means it can be used on platforms like Darwin and Windows.

package nebula

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"

	"go.uber.org/zap"
)

type udpAddr struct {
	IP   uint32
	Port uint16
}

type udpConn struct {
	*net.UDPConn
}

func NewUDPAddr(ip uint32, port uint16) *udpAddr {
	return &udpAddr{IP: ip, Port: port}
}

func NewUDPAddrFromString(s string) *udpAddr {
	p := strings.Split(s, ":")
	if len(p) < 2 {
		return nil
	}

	port, _ := strconv.Atoi(p[1])
	return &udpAddr{
		IP:   ip2int(net.ParseIP(p[0])),
		Port: uint16(port),
	}
}

func NewListener(ip string, port int, multi bool) (*udpConn, error) {
	lc := NewListenConfig(multi)
	pc, err := lc.ListenPacket(context.TODO(), "udp4", fmt.Sprintf("%s:%d", ip, port))
	if err != nil {
		return nil, err
	}
	if uc, ok := pc.(*net.UDPConn); ok {
		return &udpConn{UDPConn: uc}, nil
	}
	return nil, fmt.Errorf("Unexpected PacketConn: %T %#v", pc, pc)
}

func (ua *udpAddr) Equals(t *udpAddr) bool {
	if t == nil || ua == nil {
		return t == nil && ua == nil
	}
	return ua.IP == t.IP && ua.Port == t.Port
}

func (ua *udpAddr) Copy() udpAddr {
	return *ua
}

func (uc *udpConn) WriteTo(b []byte, addr *udpAddr) error {

	_, err := uc.UDPConn.WriteToUDP(b, &net.UDPAddr{
		IP:   int2ip(addr.IP),
		Port: int(addr.Port),
	})
	return err
}

func (uc *udpConn) LocalAddr() (*udpAddr, error) {
	a := uc.UDPConn.LocalAddr()

	switch v := a.(type) {
	case *net.UDPAddr:
		return &udpAddr{IP: ip2int(v.IP), Port: uint16(v.Port)}, nil
	default:
		return nil, fmt.Errorf("LocalAddr returned: %#v", a)
	}
}

func (u *udpConn) reloadConfig(c *Config) {
	// TODO
}

type rawMessage struct {
	Len uint32
}

func (u *udpConn) ListenOut(f *Interface) {
	plaintext := make([]byte, mtu)
	buffer := make([]byte, mtu)
	header := &Header{}
	fwPacket := &FirewallPacket{}
	udpAddr := &udpAddr{}
	nb := make([]byte, 12, 12)

	lhh := f.lightHouse.NewRequestHandler()

	for {
		// Just read one packet at a time
		n, rua, err := u.ReadFromUDP(buffer)
		if err != nil {
			l.Error("failed to read packets", zap.Error(err))
			continue
		}

		udpAddr.IP = ip2int(rua.IP)
		udpAddr.Port = uint16(rua.Port)
		f.readOutsidePackets(udpAddr, plaintext[:0], buffer[:n], header, fwPacket, lhh, nb)
	}
}

func udp2ip(addr *udpAddr) net.IP {
	return int2ip(addr.IP)
}

func udp2ipInt(addr *udpAddr) uint32 {
	return addr.IP
}

func hostDidRoam(addr *udpAddr, newaddr *udpAddr) bool {
	return !addr.Equals(newaddr)
}

func (ua *udpAddr) String() string {
	return fmt.Sprintf("%s:%v", int2ip(ua.IP), ua.Port)
}
