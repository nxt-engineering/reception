package common

type Config struct {
	Projects        *Projects
	TLD             string
	HTTPBindAddress string
	DNSBindAddress  string
	DockerEndpoint  string
}
