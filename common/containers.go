package common

import "sync"

type Containers struct {
	sync.RWMutex
	A []Container
}

// Returns all URLS which the containers expose
func (c *Containers) AllUrls() (urls map[string]string) {
	urls = make(map[string]string)

	c.RLock()
	defer c.RUnlock()
	for _, container := range c.A {
		containerUrls, err := container.AllUrls()
		if err != nil {
			continue
		}

		for from, to := range containerUrls {
			urls[from] = to
		}
	}
	return
}
