package docker

import (
	"fmt"
	"time"

	"github.com/fsouza/go-dockerclient"
)

type Client struct {
	endpoint string
}

func (thisClient Client) Launch() error {
	if thisClient.endpoint == "" {
		thisClient.endpoint = "unix:///var/run/docker.sock"
	}

	client, err := docker.NewClient(thisClient.endpoint)
	if err != nil {
		return err
	}

	containers, err := client.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return err
	}

	for _, container := range containers {
		fmt.Println("------------")
		fmt.Println("dc project: ", container.Labels["com.docker.compose.project"])
		fmt.Println("dc service: ", container.Labels["com.docker.compose.service"])
		fmt.Println("dc container number: ", container.Labels["com.docker.compose.container-number"])
		for _, port := range container.Ports {
			if port.PublicPort == 0 || port.Type != "tcp" {
				continue
			}

			fmt.Println("> ")
			fmt.Println("> PublicPort", port.PublicPort)
			fmt.Println("> IP", port.IP)
			fmt.Println("> Type", port.Type)
		}
		fmt.Println("Names: ", container.Names[0][1:])
	}

	listener := make(chan *docker.APIEvents)
	err = client.AddEventListener(listener)
	if err != nil {
		return err
	}

	defer func() {
		err = client.RemoveEventListener(listener)
		if err != nil {
			fmt.Println(err)
		}
	}()

	timeout := time.After(30 * time.Second)

	for {
		select {
		case event := <-listener:
			handleEvent(event)
		case <-timeout:
			return nil
		}
	}
	return nil
}

func handleEvent(event *docker.APIEvents) bool {
	if event.Type != "container" {
		return false
	}

	switch event.Action {
	case "start", "stop":
		fmt.Println("Action: ", event.Action)
		fmt.Println("ID: ", event.ID)
		return true
	}

	return false
}
