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

	config := &common.Config{
		BindAddress:    "localhost:8888",
		TLD:            "docker.",
		Projects:       common.NewProjects(),
		DockerEndpoint: "unix:///var/run/docker.sock",
	}

	go runHttpFrontend(config)
	go runDns(config)

	runDockerClient(config)
}

func runDns(config *common.Config) {
	handler := dns.Handler{
		Config: config,
	}

	miekg_dns.HandleFunc("docker.", handler.ServeDns)

	addr := "localhost:5300"
	fmt.Printf("Listening on '%v' for DNS requests.\n", addr)

	srv := &miekg_dns.Server{Addr: addr, Net: "udp"}
	err := srv.ListenAndServe()
	if err != nil {
		panic(err)
	}
}

func runDockerClient(config *common.Config) {
	client := docker.Client{
		Config: config,
	}
	err := client.Launch()
	if err != nil {
		panic(err)
	}
}

func runHttpFrontend(config *common.Config) {
	frontend := &net_http.Server{
		Addr: config.BindAddress,
		Handler: http.BackendHandler{
			Config: config,
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
