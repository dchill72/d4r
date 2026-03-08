package docker

import "time"

type Container struct {
	ID      string
	FullID  string
	Name    string
	Image   string
	Status  string
	State   string // "running", "exited", "paused", "created", etc.
	Ports   string
	Created time.Time
}

type Volume struct {
	Name       string
	Driver     string
	Mountpoint string
	Scope      string
	CreatedAt  string
	Size       int64 // bytes, -1 if unknown
	RefCount   int64 // number of containers referencing it
	Labels     map[string]string
}

type Network struct {
	ID         string
	Name       string
	Driver     string
	Scope      string
	Internal   bool
	IPv6       bool
	Subnet     string
	Gateway    string
	Containers []string
	Labels     map[string]string
}

type Image struct {
	ID      string // short 12-char
	FullID  string
	Tags    []string
	Size    int64
	Created time.Time
}
