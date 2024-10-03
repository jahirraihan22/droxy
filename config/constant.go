package config

import "github.com/docker/docker/api/types"

var ContainerCache = make(map[string]types.Container)
var ImageCache []string
