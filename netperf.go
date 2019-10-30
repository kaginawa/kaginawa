package main

import (
	"net"
	"os"
	"sort"
	"time"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

const (
	pingTries   = 3
	pingTimeout = 4 * time.Second
	pingBytes   = 56
)

func pingLatency(target string) (float64, error) {
	payload := make([]byte, pingBytes)
	ip, err := net.ResolveIPAddr("ip4", target)
	if err != nil {
		return -1, err
	}
	results := make([]float64, pingTries)
	for i := 0; i < pingTries; i++ {
		elapsed, err := doPing(payload, i, ip)
		if err != nil {
			return -1, err
		}
		results[i] = elapsed
	}
	sort.Float64s(results)
	return results[1], nil // pick median
}

func doPing(payload []byte, i int, ip net.Addr) (float64, error) {
	msg := icmp.Message{
		Type: ipv4.ICMPTypeEcho,
		Code: 0,
		Body: &icmp.Echo{ID: os.Getpid() & 0xffff, Seq: i, Data: payload},
	}
	raw, err := msg.Marshal(nil)
	if err != nil {
		return -1, err
	}
	begin := time.Now()
	conn, err := icmp.ListenPacket("ip4:icmp", "0.0.0.0")
	if err != nil {
		return -1, err
	}
	defer safeClose(conn, "ping connection")
	if _, err := conn.WriteTo(raw, ip); err != nil {
		return -1, err
	}
	if err := conn.SetReadDeadline(begin.Add(pingTimeout)); err != nil {
		return -1, err
	}
	buf := make([]byte, 1500)
	n, _, err := conn.ReadFrom(buf)
	if err != nil {
		return -1, err
	}
	elapsed := float64(time.Since(time.Now()).Nanoseconds()) / 1000000
	rm, err := icmp.ParseMessage(ipv4.ICMPTypeEcho.Protocol(), buf[:n])
	if err != nil || rm.Type != ipv4.ICMPTypeEchoReply {
		return -1, err
	}
	return elapsed, nil
}
