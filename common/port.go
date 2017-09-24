package common

import (
	"github.com/fsouza/go-dockerclient"
)

type Port struct {
	PrivatePort  uint32
	LocalPort    uint32
	LocalAddress string
	Protocol     Protocol
	IsExposed    bool
}

func PortFromApiPort(apiPort docker.APIPort) *Port {
	var proto Protocol

	switch apiPort.Type {
	case "tcp":
		proto = TCP
	case "udp":
		proto = UDP
	default:
		panic("Unknown Port Type")
	}

	return &Port{
		IsExposed:    apiPort.PublicPort != 0,
		LocalPort:    uint32(apiPort.PublicPort),
		LocalAddress: apiPort.IP,
		PrivatePort:  uint32(apiPort.PrivatePort),
		Protocol:     proto,
	}
}
