package common

import "sync"

// maps Host (from request header) to destination Host
// always acquire the lock, first!
type HostToHostMap struct {
	sync.RWMutex
	M map[string]string
}
