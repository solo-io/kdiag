//go:build !linux

package sockets

import (
	"context"
	"errors"
)

func GetListeningPorts(ctx context.Context) (map[int]ProcessSockets, error) {
	return nil, errors.New("not implemented")
}
