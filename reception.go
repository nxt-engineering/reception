package main

import (
	"fmt"

	net_http "net/http"

	miekg_dns "github.com/miekg/dns"
	"github.com/ninech/reception/common"
	"github.com/ninech/reception/dns"
	"github.com/ninech/reception/docker"
	"github.com/ninech/reception/http"
)

func main() {
	fmt.Println("(c) 2017 Nine Internet Solutions AG")

	hostMap := &common.HostToHostMap{
		M: make(map[string]string),
	}

	hostMap.Lock()
	hostMap.M["localhost"] = "google.com:80"
	hostMap.Unlock()

	go runHttpFrontend(hostMap)
	go runDns(hostMap)

	runDockerClient(hostMap)
}

func runDns(hostMap *common.HostToHostMap) {
	dnsHandler := dns.Handler{
		HostMap: hostMap,
	}

	miekg_dns.HandleFunc("docker.", dnsHandler.ServeDns)

	addr := "localhost:5300"
	fmt.Printf("Listening on '%v' for DNS requests.\n", addr)

	srv := &miekg_dns.Server{Addr: addr, Net: "udp"}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func runDockerClient(hostMap *common.HostToHostMap) {
	client := docker.Client{
		HostMap: hostMap,
	}
	err := client.Launch()
	if err != nil {
		panic(err)
	}
}

func runHttpFrontend(hostMap *common.HostToHostMap) {
	frontend := &net_http.Server{
		Addr: "localhost:8888",
		Handler: http.BackendHandler{
			HostMap: hostMap,
		},
	}

	fmt.Printf("Listening on '%v' for HTTP traffic.\n", frontend.Addr)

	err := frontend.ListenAndServe()
	if err != nil {
		panic(err)
	} else {
		defer frontend.Close()
	}
}
