package docker

import (
	"fmt"
	"time"

	"strconv"

	"github.com/fsouza/go-dockerclient"
	"github.com/ninech/reception/common"
)

type Container struct {
	ID              string
	Name            string
	PublicPorts     []docker.APIPort
	Project         string
	Service         string
	ContainerNumber string
	HttpPort        int64
	IsMain          bool
}

type Client struct {
	Endpoint     string
	HostMap      *common.HostToHostMap
	containerMap map[string]Container
}

// private_port.service.project.docker:public_port
// service.project.docker:public_port
// project.docker:port // reception.main | app | 80 | 8080

func (client *Client) Launch() error {
	if client.Endpoint == "" {
		client.Endpoint = "unix:///var/run/docker.sock"
	}

	dockerClient, err := docker.NewClient(client.Endpoint)
	if err != nil {
		return err
	}

	containers, err := dockerClient.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return err
	}

	buildContainerMap(containers, client)

	if len(client.containerMap) == 0 {
		return nil
	}

	/*
		Start of building Hostname Map
	*/
	var mainProjectContainer, appProjectContainer *Container

	client.HostMap.Lock()
	defer client.HostMap.Unlock()
	for _, container := range client.containerMap {
		if container.IsMain {
			if mainProjectContainer == nil {
				mainProjectContainer = &container
			} else {
				//TODO write warning: more than 1 "main"
			}
		}

		if container.Name == "app" && appProjectContainer == nil {
			appProjectContainer = &container
		}

		mainServiceHostSet := false
		for _, port := range container.PublicPorts {
			// private_port.service.project.docker:public_port
			backendHostname := fmt.Sprintf("%v:%v", port.IP, port.PublicPort)

			frontendPortHostname := fmt.Sprintf(
				"%v.%v.%v.docker:%v",
				port.PrivatePort,
				container.Service,
				container.Project,
				port.PublicPort)
			client.HostMap.M[frontendPortHostname] = backendHostname

			// service.project.docker:public_port
			if !mainServiceHostSet {
				switch port.PrivatePort {
				case container.HttpPort:
					// this is the designated http port set by the user
				case 80, 8080, 3000:
					// these are common http ports
				default:
					continue
				}

				frontendServiceHostname := fmt.Sprintf(
					"%v.%v.docker:%v",
					container.Service,
					container.Project,
					port.PublicPort)

				client.HostMap.M[frontendServiceHostname] = backendHostname
				mainServiceHostSet = true
			}
		}

		if mainProjectContainer == nil {
			if appProjectContainer != nil {
				mainProjectContainer = appProjectContainer
			} else {
				return nil
			}
		}

		// project.docker:public_port
		var theMainPort, aCommonHttpPort *docker.APIPort
		lowestPort := &mainProjectContainer.PublicPorts[0]
		for _, port := range mainProjectContainer.PublicPorts {
			if port.PrivatePort < lowestPort.PrivatePort {
				lowestPort = &port
			}

			switch port.PrivatePort {
			case container.HttpPort:
				theMainPort = &port
				break
			case 80, 8080, 3000:
				aCommonHttpPort = &port
			}
		}

		var thePort *docker.APIPort
		if theMainPort != nil {
			thePort = theMainPort
		} else if aCommonHttpPort != nil {
			thePort = aCommonHttpPort
		} else {
			thePort = lowestPort
		}

		frontendProjectHostname := fmt.Sprintf(
			"%v.%v.docker:%v",
			container.Service,
			container.Project,
			thePort.PublicPort)
		backendHostname := fmt.Sprintf("%v:%v", thePort.IP, thePort.PublicPort)
		client.HostMap.M[frontendProjectHostname] = backendHostname
	}

	listener := make(chan *docker.APIEvents)
	err = dockerClient.AddEventListener(listener)
	if err != nil {
		return err
	}

	defer func() {
		err = dockerClient.RemoveEventListener(listener)
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

func buildContainerMap(containers []docker.APIContainers, client *Client) {
	for _, container := range containers {
		if !hasPublicTcpPorts(container.Ports) {
			continue
		}

		_, ok := container.Labels["reception.main"]
		httpPort, err := strconv.ParseInt(container.Labels["reception.http-port"], 10, 64)
		if err != nil {
			httpPort = 8080
		}

		client.containerMap[container.ID] = Container{
			ID:              container.ID,
			Name:            container.Names[0][1:],
			PublicPorts:     filterPublicTcpPorts(container.Ports),
			Project:         container.Labels["com.docker.compose.project"],
			Service:         container.Labels["com.docker.compose.service"],
			ContainerNumber: container.Labels["com.docker.compose.container-number"],
			HttpPort:        httpPort,
			IsMain:          ok,
		}
	}
}

func hasPublicTcpPorts(apiPorts []docker.APIPort) bool {
	for _, port := range apiPorts {
		if port.PublicPort != 0 && port.Type != "tcp" {
			return true
		}
	}
	return false
}

func filterPublicTcpPorts(apiPorts []docker.APIPort) (publicPorts []docker.APIPort) {
	for _, port := range apiPorts {
		if port.PublicPort == 0 || port.Type != "tcp" {
			continue
		}
		publicPorts = append(publicPorts, port)
	}
	return
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
