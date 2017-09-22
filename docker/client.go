package docker

import (
	"fmt"
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
	containerMap *map[string]Container
	dockerClient *docker.Client
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
	client.dockerClient = dockerClient

	containerMap := make(map[string]Container)
	client.containerMap = &containerMap

	client.updateMappings()

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

	for {
		event := <-listener
		err = client.handleEvent(event)
		if err != nil {
			return err
		}
	}
	return nil
}

// re-reads the containers and re-builds the hostToHost Mapping
func (client *Client) updateMappings() error {

	err := client.updateContainers()
	if err != nil {
		return err
	}

	err = client.updateHostMap()
	if err != nil {
		return err
	}
	return nil
}

// updates the map which maps containers names to their public ports
func (client *Client) updateHostMap() error {
	var mainProjectContainer, appProjectContainer *Container

	client.HostMap.Lock()
	defer client.HostMap.Unlock()

	// clean the map
	for key := range client.HostMap.M {
		delete(client.HostMap.M, key)
	}

	containerMap := *client.containerMap

	// add the appropriate names for any container to the map
	for _, container := range containerMap {

		if container.IsMain {
			if mainProjectContainer == nil {
				mainProjectContainer = &container
			} else {
				fmt.Errorf(
					"More than one container with 'reception.main' label, at least '%v' (chosen) and '%v' (ignoring).",
					mainProjectContainer.Name,
					container.Name,
				)
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
				"%v.%v.%v.docker",
				port.PrivatePort,
				container.Service,
				container.Project)
			client.HostMap.M[frontendPortHostname] = backendHostname
			fmt.Printf("Added route from '%v' to '%v'.\n", frontendPortHostname, backendHostname)

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
					"%v.%v.docker",
					container.Service,
					container.Project)
				client.HostMap.M[frontendServiceHostname] = backendHostname
				fmt.Printf("Added route from '%v' to '%v'.\n", frontendServiceHostname, backendHostname)

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
			"%v.%v.docker",
			container.Service,
			container.Project)
		backendHostname := fmt.Sprintf("%v:%v", thePort.IP, thePort.PublicPort)
		client.HostMap.M[frontendProjectHostname] = backendHostname
		fmt.Printf("Added route from '%v' to '%v'.\n", frontendProjectHostname, backendHostname)
	}

	return nil
}

// updates the list of containers
func (client *Client) updateContainers() error {
	containers, err := client.dockerClient.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return err
	}

	// clean old content
	containerMap := *client.containerMap
	for k := range containerMap {
		delete(containerMap, k)
	}

	// add all containers to the map
	for _, container := range containers {
		client.addToContainerMap(container)
	}
	return nil
}

// adds a container to the container map
func (client *Client) addToContainerMap(container docker.APIContainers) {
	if !hasPublicTcpPorts(container.Ports) {
		return
	}

	_, ok := container.Labels["reception.main"]
	httpPort, err := strconv.ParseInt(container.Labels["reception.http-port"], 10, 64)
	if err != nil {
		httpPort = 8080
	}

	publicTcpPorts := filterPublicTcpPorts(container.Ports)

	containerMap := *client.containerMap
	containerMap[container.ID] = Container{
		ID:              container.ID,
		Name:            container.Names[0][1:],
		PublicPorts:     publicTcpPorts,
		Project:         container.Labels["com.docker.compose.project"],
		Service:         container.Labels["com.docker.compose.service"],
		ContainerNumber: container.Labels["com.docker.compose.container-number"],
		HttpPort:        httpPort,
		IsMain:          ok,
	}
}

// returns true, if any of the ports are exposed to the host machine
func hasPublicTcpPorts(apiPorts []docker.APIPort) bool {
	for _, port := range apiPorts {
		if port.PublicPort != 0 && port.Type == "tcp" {
			return true
		}
	}
	return false
}

// of the given ports it only returns those who are exposed to the host machine
func filterPublicTcpPorts(apiPorts []docker.APIPort) (publicPorts []docker.APIPort) {
	for _, port := range apiPorts {
		if port.PublicPort == 0 || port.Type != "tcp" {
			continue
		}
		publicPorts = append(publicPorts, port)
	}
	return
}

// handles an event emitted by Docker
func (client *Client) handleEvent(event *docker.APIEvents) error {
	if event.Type != "container" {
		return nil
	}

	switch event.Action {
	case "start", "stop":

		err := client.updateMappings()
		if err != nil {
			return err
		}
	}
	return nil
}
