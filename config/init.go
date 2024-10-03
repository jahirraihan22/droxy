package config

import (
	"github.com/docker/docker/client"
	"sync"
)

var once sync.Once
var cli *client.Client
var dcrErr error

func getClient() (*client.Client, error) {
	c, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return c, nil
}

func InitiateClientSet() error {
	once.Do(func() {
		cli, dcrErr = getClient()
	})
	if dcrErr != nil {
		return dcrErr
	}
	return nil
}

func DockerClient() *client.Client {
	return cli
}
