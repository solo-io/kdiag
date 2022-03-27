package sockets

import (
	"net/netip"

	"github.com/samber/lo"
)

type ProcessSockets struct {
	Pid     int
	Sockets []*Socket
}

func (p *ProcessSockets) Ports() []uint16 {
	return lo.Map(p.Sockets, func(s *Socket, _ int) uint16 {
		return uint16(s.ID.SourcePort)
	})
}

type SocketID struct {
	SourcePort      uint16
	DestinationPort uint16
	Source          netip.Addr
	Destination     netip.Addr
	Interface       uint32
	Cookie          [2]uint32
}

// Socket represents a netlink socket.
type Socket struct {
	Family  uint8
	State   uint8
	Timer   uint8
	Retrans uint8
	ID      SocketID
	Expires uint32
	RQueue  uint32
	WQueue  uint32
	UID     uint32
	INode   uint32
}
