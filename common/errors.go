package common

import "fmt"

type NoExposedPorts struct {
	container *Container
}

func (e NoExposedPorts) Error() string {
	return fmt.Sprintf("No exosed ports on container '%v'", e.container.ID)
}
