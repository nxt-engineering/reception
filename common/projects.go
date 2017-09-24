package common

import "sync"

type Projects struct {
	sync.RWMutex
	M map[string]*Project
}

func NewProjects() *Projects {
	return &Projects{
		M: make(map[string]*Project),
	}
}

func (ps *Projects) GetOrCreate(name string) (project *Project) {
	ps.RLock()
	projects := *ps
	project, existing := projects.M[name]
	ps.RUnlock()

	if !existing {
		project = &Project{
			Name: name,
		}

		ps.Lock()
		projects.M[name] = project
		ps.Unlock()
	}

	return
}

func (ps *Projects) AllUrls() (urls map[string]string) {
	ps.RLock()
	defer ps.RUnlock()

	urls = make(map[string]string)

	for _, project := range ps.M {
		allUrls := project.AllUrls()
		for from, to := range allUrls {
			urls[from] = to
		}
	}
	return
}
