package docker

import (
	"fmt"

	"github.com/fsouza/go-dockerclient"
	"github.com/ninech/reception/common"
)

type Client struct {
	Endpoint     string
	Config       *common.Config
	dockerClient *docker.Client
}

func (client *Client) Launch() error {
	if client.Endpoint == "" {
		client.Endpoint = client.Config.DockerEndpoint
	}

	dockerClient, err := docker.NewClient(client.Endpoint)
	if err != nil {
		return err
	}
	client.dockerClient = dockerClient

	err = client.updateContainers()
	if err != nil {
		return err
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

	for {
		event := <-listener
		err = client.handleEvent(event)
		if err != nil {
			return err
		}
	}
}

// handles an event emitted by Docker
func (client *Client) handleEvent(event *docker.APIEvents) error {
	if event == nil {
		return nil
	}

	if event.Type != "container" {
		return nil
	}

	switch event.Action {
	case "start", "stop":

		err := client.updateContainers()
		if err != nil {
			return err
		}
	}
	return nil
}

// updates the list of containers
func (client *Client) updateContainers() error {
	containers, err := client.dockerClient.ListContainers(docker.ListContainersOptions{All: false})
	if err != nil {
		return err
	}

	client.removeAllProjects()
	client.addAllContainers(containers)

	return nil
}

func (client *Client) addAllContainers(containers []docker.APIContainers) {
	for _, container := range containers {
		common.ContainerFromApiContainer(container, client.Config.Projects)
	}
}

func (client *Client) removeAllProjects() {
	client.Config.Projects.Lock()
	defer client.Config.Projects.Unlock()

	for k := range client.Config.Projects.M {
		delete(client.Config.Projects.M, k)
	}
}
