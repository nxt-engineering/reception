package common

import (
	"fmt"
	"strconv"

	"github.com/fsouza/go-dockerclient"
)

type Container struct {
	ID              string
	Name            string
	Ports           []*Port
	Project         *Project
	Service         string
	ContainerNumber string
	HttpPort        *Port
	IsMain          bool
}

// Urls for all exposed ports; does not include the MainUrl
// Format: $PORT.$CONTAINER.$SERVICE.$PROJECT -> $LOCAL_ADDR:$LOCAL_PORT
func (container *Container) Urls() (urlMapping map[string]string, err error) {
	exposedPorts := container.exposedPorts()
	if len(exposedPorts) == 0 {
		return urlMapping, NoExposedPorts{container}
	}

	urlMapping = make(map[string]string)

	for _, port := range exposedPorts {
		if !port.IsExposed {
			continue
		}

		to := fmt.Sprintf("%v:%v", port.LocalAddress, port.LocalPort)
		from := fmt.Sprintf("%v.%v.%v", port.PrivatePort, container.Service, container.Project.Name)

		urlMapping[from] = to
	}

	return
}

// The main URL for this container
// Format: $CONTAINER.$SERVICE.$PROJECT -> $LOCAL_ADDR:$LOCAL_PORT
func (container *Container) MainUrl() (from, to string, err error) {
	httpPort, err := container.MainExposedPort()
	if err != nil {
		return
	}

	to = fmt.Sprintf("%v:%v", httpPort.LocalAddress, httpPort.LocalPort)
	from = fmt.Sprintf("%v.%v", container.Service, container.Project.Name)

	return
}

func (container *Container) AllUrls() (urlMapping map[string]string, err error) {
	if !container.HasExposedTCPPorts() {
		return urlMapping, NoExposedPorts{container}
	}

	urlMapping, _ = container.Urls()

	from, to, _ := container.MainUrl()
	urlMapping[from] = to

	return
}

func (container *Container) MainExposedPort() (port *Port, err error) {
	exposedPorts := container.exposedPorts()
	if len(exposedPorts) == 0 {
		return port, NoExposedPorts{container}
	}

	port = container.HttpPort
	if port != nil {
		return
	}

	// find lowest exposed port
	port = exposedPorts[0]
	for _, exposedPort := range exposedPorts {
		if exposedPort.PrivatePort < port.PrivatePort {
			port = exposedPort
		}
	}
	return
}

func ContainerFromApiContainer(apiContainer docker.APIContainers, projects *Projects) {
	var commonContainer Container

	_, mainLabelPresent := apiContainer.Labels["reception.main"]
	httpPort, err := strconv.ParseUint(apiContainer.Labels["reception.http-port"], 10, 32)
	if err != nil {
		httpPort = 8080
	}

	project := projects.GetOrCreate(apiContainer.Labels["com.docker.compose.project"])
	commonContainer = Container{
		ID:              apiContainer.ID,
		Name:            apiContainer.Names[0][1:],
		Ports:           make([]*Port, len(apiContainer.Ports)),
		Project:         project,
		Service:         apiContainer.Labels["com.docker.compose.service"],
		ContainerNumber: apiContainer.Labels["com.docker.compose.container-number"],
		IsMain:          mainLabelPresent,
	}

	for i, port := range apiContainer.Ports {
		commonPort := PortFromApiPort(port)
		commonContainer.Ports[i] = commonPort

		switch commonPort.PrivatePort {
		case uint32(httpPort), 80, 8080, 3000:
			commonContainer.HttpPort = commonContainer.Ports[i]
		}
	}

	project.Containers.Lock()
	defer project.Containers.Unlock()
	project.Containers.A = append(project.Containers.A, commonContainer)
}

// true if there are any exposed ports on this host
func (container *Container) HasExposedTCPPorts() bool {
	for _, port := range container.Ports {
		if port.IsExposed && port.Protocol == TCP {
			return true
		}
	}
	return false
}

func (container *Container) exposedPorts() (publicPorts []*Port) {
	for _, port := range container.Ports {
		if !port.IsExposed {
			continue
		}

		publicPorts = append(publicPorts, port)
	}
	return
}
