package iphash

import (
	"context"
	"fmt"
	"hash/crc32"
	"net"
	"strings"

	"github.com/PaperMan11/goim/pkg/protocol/constant"
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

type ipHashPicker struct {
	subConns []balancer.SubConn
}

func newIPHashPicker(subConns []balancer.SubConn) balancer.Picker {
	return &ipHashPicker{
		subConns: subConns,
	}
}

func (p *ipHashPicker) Pick(info balancer.PickInfo) (balancer.PickResult, error) {
	if len(p.subConns) == 0 {
		return balancer.PickResult{}, balancer.ErrNoSubConnAvailable
	}

	if len(p.subConns) == 1 {
		return balancer.PickResult{SubConn: p.subConns[0]}, nil
	}

	ip := extractIPFromContext(info.Ctx)
	if ip == "" {
		return balancer.PickResult{SubConn: p.subConns[0]}, nil
	}

	index := hashIP(ip) % len(p.subConns)

	return balancer.PickResult{SubConn: p.subConns[index]}, nil
}

func extractIPFromContext(ctx context.Context) string {
	if md, ok := peer.FromContext(ctx); ok {
		if addr, ok := md.Addr.(*net.TCPAddr); ok {
			return addr.IP.String()
		}
	}

	return extractIPFromMetadata(ctx)
}

func extractIPFromMetadata(ctx context.Context) string {
	md, ok := metadata.FromOutgoingContext(ctx)
	if !ok {
		return ""
	}

	if values := md.Get(constant.ClientIP); len(values) > 0 {
		ip := values[0]
		if idx := strings.Index(ip, ","); idx != -1 {
			ip = strings.TrimSpace(ip[:idx])
		}
		if net.ParseIP(ip) != nil {
			return ip
		}
	}

	return ""
}

func hashIP(ip string) int {
	if strings.Contains(ip, ":") {
		parsedIP := net.ParseIP(ip)
		if parsedIP != nil {
			ipv6 := parsedIP.To16()
			return int(crc32.ChecksumIEEE(ipv6))
		}
	}

	parts := strings.Split(ip, ".")
	if len(parts) == 4 {
		var ipInt uint32
		for i := 0; i < 4; i++ {
			var part uint32
			fmt.Sscanf(parts[i], "%d", &part)
			ipInt |= part << uint(8*(3-i))
		}
		return int(ipInt)
	}

	return int(crc32.ChecksumIEEE([]byte(ip)))
}
