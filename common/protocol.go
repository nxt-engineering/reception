package common

type Protocol int

const (
	// https://en.wikipedia.org/wiki/List_of_IP_protocol_numbers
	TCP Protocol = 0x06 // 6
	UDP Protocol = 0x11 // 17
)
