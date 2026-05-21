// Package netutil exposes small helpers for talking to / discovering LAN
// peers. The Times Frame polls our HTTP endpoints itself, so we need to
// pass it URLs that resolve from its perspective (i.e. a routable LAN
// address, not 127.0.0.1).
package netutil

import (
	"fmt"
	"net"
)

// LANAddress returns the IP this process would use when sending packets to
// the public internet. On a typical home network that's the LAN interface
// the Times Frame can reach back to. We do NOT actually send a packet —
// `net.Dial("udp", ...)` on an unconnected UDP socket just resolves the
// local source address.
func LANAddress() (net.IP, error) {
	conn, err := net.Dial("udp", "8.8.8.8:53")
	if err != nil {
		return nil, fmt.Errorf("resolve local LAN address: %w", err)
	}
	defer conn.Close()
	addr, ok := conn.LocalAddr().(*net.UDPAddr)
	if !ok {
		return nil, fmt.Errorf("unexpected local addr type %T", conn.LocalAddr())
	}
	return addr.IP, nil
}
