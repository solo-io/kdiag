package sockets

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"net/netip"
	"path/filepath"
	"syscall"

	ps "github.com/mitchellh/go-ps"
	"github.com/vishvananda/netlink/nl"
	"golang.org/x/sys/unix"
)

type inodeAndPort struct {
	inode int64
	port  int
}

func GetListeningPorts(ctx context.Context) (map[int]ProcessSockets, error) {
	listeningsokcets, err := SocketListen()
	if err != nil {
		return nil, err
	}

	sockets, err := GetProcessSockets()
	if err != nil {
		return nil, err
	}
	ret := map[int]ProcessSockets{}

	for pid, inodes := range sockets {
		ps := ProcessSockets{Pid: pid}
		for _, socket := range listeningsokcets {
			for _, pidsock := range inodes {
				if uint64(socket.INode) == pidsock {
					ps.Sockets = append(ps.Sockets, socket)
				}
			}
		}
		ret[ps.Pid] = ps
	}

	return ret, nil
}

func GetProcessSockets() (map[int][]uint64, error) {
	processList, err := ps.Processes()
	if err != nil {
		return nil, fmt.Errorf("could not get process list: %w", err)
	}

	var socketsmap = make(map[int][]uint64)
	for _, p := range processList {
		pid := p.Pid()
		sockets, err := GetSocketInodesFor(pid)
		if err != nil {
			return nil, err
		}

		for _, sock := range sockets {
			socketsmap[pid] = append(socketsmap[pid], sock)
		}
	}

	return socketsmap, nil
}

func GetSocketInodesFor(pid int) ([]uint64, error) {
	directory := fmt.Sprintf("/proc/%d/fd/", pid)
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}
	var inodes []uint64
	for _, f := range files {
		if f.IsDir() {
			continue
		}
		abspath := filepath.Join(directory, f.Name())

		var stat syscall.Stat_t
		if err := syscall.Stat(abspath, &stat); err != nil {
			continue
		}
		issock := (stat.Mode & syscall.S_IFSOCK) == syscall.S_IFSOCK
		if !issock {
			continue
		}

		inodes = append(inodes, stat.Ino)
	}
	return inodes, nil
}

const (
	sizeofSocketID      = 0x30
	sizeofSocketRequest = sizeofSocketID + 0x8
	sizeofSocket        = sizeofSocketID + 0x18
)

type readBuffer struct {
	Bytes []byte
	pos   int
	err   error
}

func (b *readBuffer) Read() byte {
	if b.err != nil {
		return 0
	}
	if b.pos > len(b.Bytes) {
		b.err = fmt.Errorf("socket data short read (%d); want %d", len(b.Bytes), b.pos)
		return 0
	}
	c := b.Bytes[b.pos]
	b.pos++
	return c
}

func (b *readBuffer) Next(n int) []byte {
	if b.err != nil {
		return nil
	}

	if b.pos+n > len(b.Bytes) {
		b.err = fmt.Errorf("socket data short read (%d); want %d", len(b.Bytes), b.pos+n)
		return nil
	}

	s := b.Bytes[b.pos : b.pos+n]
	b.pos += n
	return s
}

func (s *Socket) deserialize(b []byte) error {
	if len(b) < sizeofSocket {
		return fmt.Errorf("socket data short read (%d); want %d", len(b), sizeofSocket)
	}
	rb := readBuffer{Bytes: b}
	s.Family = rb.Read()
	s.State = rb.Read()
	s.Timer = rb.Read()
	s.Retrans = rb.Read()
	s.ID.SourcePort = networkOrder.Uint16(rb.Next(2))
	s.ID.DestinationPort = networkOrder.Uint16(rb.Next(2))

	if s.Family == unix.AF_INET6 {
		s.ID.Source, _ = netip.AddrFromSlice(rb.Next(16))
		s.ID.Destination, _ = netip.AddrFromSlice(rb.Next(16))
	} else {
		s.ID.Source, _ = netip.AddrFromSlice(rb.Next(4))
		rb.Next(12)
		s.ID.Destination, _ = netip.AddrFromSlice(rb.Next(4))
		rb.Next(12)
	}

	s.ID.Interface = native.Uint32(rb.Next(4))
	s.ID.Cookie[0] = native.Uint32(rb.Next(4))
	s.ID.Cookie[1] = native.Uint32(rb.Next(4))
	s.Expires = native.Uint32(rb.Next(4))
	s.RQueue = native.Uint32(rb.Next(4))
	s.WQueue = native.Uint32(rb.Next(4))
	s.UID = native.Uint32(rb.Next(4))
	s.INode = native.Uint32(rb.Next(4))
	return rb.err
}

type socketRequest struct {
	Family   uint8
	Protocol uint8
	Ext      uint8
	pad      uint8
	States   uint32
	ID       SocketID
}

type writeBuffer struct {
	Bytes []byte
	pos   int
}

func (b *writeBuffer) Write(c byte) {
	b.Bytes[b.pos] = c
	b.pos++
}

func (b *writeBuffer) Next(n int) []byte {
	s := b.Bytes[b.pos : b.pos+n]
	b.pos += n
	return s
}

var (
	native       = nl.NativeEndian()
	networkOrder = binary.BigEndian
)

func (r *socketRequest) Serialize() []byte {
	b := writeBuffer{Bytes: make([]byte, sizeofSocketRequest)}
	b.Write(r.Family)
	b.Write(r.Protocol)
	b.Write(r.Ext)
	b.Write(r.pad)
	native.PutUint32(b.Next(4), r.States)
	networkOrder.PutUint16(b.Next(2), r.ID.SourcePort)
	networkOrder.PutUint16(b.Next(2), r.ID.DestinationPort)
	copy(b.Next(16), r.ID.Source.AsSlice())
	copy(b.Next(16), r.ID.Destination.AsSlice())
	native.PutUint32(b.Next(4), r.ID.Interface)
	native.PutUint32(b.Next(4), r.ID.Cookie[0])
	native.PutUint32(b.Next(4), r.ID.Cookie[1])
	return b.Bytes
}

func (r *socketRequest) Len() int { return sizeofSocketRequest }

const (
	LISTEN = 1024
)

func SocketListen() ([]*Socket, error) {

	s, err := nl.Subscribe(syscall.NETLINK_INET_DIAG)
	if err != nil {
		return nil, err
	}
	defer s.Close()
	req := nl.NewNetlinkRequest(nl.SOCK_DIAG_BY_FAMILY, syscall.NLM_F_REQUEST|syscall.NLM_F_DUMP)
	req.AddData(&socketRequest{
		Family:   syscall.AF_INET,
		Protocol: syscall.IPPROTO_TCP,
		States:   LISTEN,
		ID:       SocketID{},
	})
	s.Send(req)
	msgs, _, err := s.Receive()
	if err != nil {
		return nil, err
	}
	if len(msgs) == 0 {
		return nil, errors.New("no message nor error from netlink")
	}
	var sockets []*Socket
	for _, msg := range msgs {
		if msg.Header.Type == nl.SOCK_DIAG_BY_FAMILY {

			sock := &Socket{}
			if err := sock.deserialize(msg.Data); err != nil {
				continue
			}
			sockets = append(sockets, sock)
		} else if msg.Header.Type == syscall.NLMSG_ERROR {
			errval := native.Uint32(msg.Data[:4])
			return nil, fmt.Errorf("netlink error: %d", -errval)
		}

	}
	return sockets, nil
}
