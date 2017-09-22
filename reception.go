package main

import (
	"fmt"

	net_http "net/http"

	"github.com/ninech/reception/common"
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

	runDockerClient(hostMap)
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
			HostMapping: hostMap,
		},
	}

	fmt.Println("Starting to listen on ", frontend.Addr)

	err := frontend.ListenAndServe()
	if err != nil {
		panic(err)
	} else {
		defer frontend.Close()
	}
}
