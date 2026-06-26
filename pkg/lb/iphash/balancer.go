package iphash

import (
	"google.golang.org/grpc/balancer"
	"google.golang.org/grpc/balancer/base"
)

const Name = "iphash"

func init() {
	balancer.Register(newBuilder())
}

func newBuilder() balancer.Builder {
	return base.NewBalancerBuilder(Name, &ipHashPickerBuilder{}, base.Config{HealthCheck: true})
}

type ipHashPickerBuilder struct{}

func (*ipHashPickerBuilder) Build(info base.PickerBuildInfo) balancer.Picker {
	if len(info.ReadySCs) == 0 {
		return base.NewErrPicker(balancer.ErrNoSubConnAvailable)
	}

	var subConns []balancer.SubConn
	for sc := range info.ReadySCs {
		subConns = append(subConns, sc)
	}

	return newIPHashPicker(subConns)
}
