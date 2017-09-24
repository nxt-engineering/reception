package common

import "fmt"

type Project struct {
	Name       string
	Containers Containers
}

// The entrypoint container for this project.
func (p Project) MainContainer() *Container {
	p.Containers.RLock()
	defer p.Containers.RUnlock()

	if len(p.Containers.A) == 0 {
		return nil
	}

	for _, container := range p.Containers.A {
		if container.IsMain {
			return &container
		}
	}

	return &p.Containers.A[0]
}

// URL to MainContainer of this Project
func (p Project) Url() (from, to string, err error) {
	mainContainer := p.MainContainer()

	port, err := mainContainer.MainExposedPort()
	if err != nil {
		return
	}

	to = fmt.Sprintf("%v:%v", port.LocalAddress, port.LocalPort)
	from = fmt.Sprintf("%v.%v.%v", port.PrivatePort, mainContainer.Service, mainContainer.Project.Name)

	return
}

// Returns any URL that this project or it's container expose
func (p Project) AllUrls() (urls map[string]string) {
	urls = p.Containers.AllUrls()

	from, to, err := p.Url()
	if err != nil {
		urls[from] = to
	}

	return
}
