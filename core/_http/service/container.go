package service

import (
	"context"
	"droxy/config"
	"fmt"
	"github.com/docker/docker/api/types/image"

	"github.com/docker/docker/api/types/container"
	"log"
	"strings"
)

type Container struct {
	Name  string `json:"name"`
	Image string `json:"image"`
	Tag   string `json:"tag"`
}

func (c Container) CreateContainer(name string, imageName string, tag string) error {
	containerConfig := new(container.Config)
	found := false
	for _, img := range config.ImageCache {
		if img == imageName+":"+tag {
			found = true
			containerConfig.Image = img
			break
		} else {
			continue
		}
	}

	if !found {
		containerConfig.Image = imageName + ":" + tag
		log.Println("[Info] Image not found locally, pulling from hub...")
		_, err := config.DockerClient().ImagePull(context.Background(), imageName+":"+tag, image.PullOptions{})
		if err != nil {
			log.Println("[Error] Container image pull failed ", err)
			return err
		}
		// update cache image
		CacheImage()
	}

	resp, err := config.DockerClient().ContainerCreate(context.Background(), containerConfig, nil, nil, nil, name)
	if err != nil {
		log.Println("[Error] creating container ", err)
		return err
	}

	fmt.Println("Container created successfully with ID:", resp.ID)

	if err = config.DockerClient().ContainerStart(context.Background(), resp.ID, container.StartOptions{}); err != nil {
		log.Println("[Error] starting container", err)
		return err
	}

	log.Println("Container started successfully")

	CacheContainer()
	return nil
}

func CacheContainer() {
	containers, err := config.DockerClient().ContainerList(context.Background(), container.ListOptions{All: true})
	if err != nil {
		log.Println("[Error]: Something wrong with get Container info ", err)
	}

	log.Println("[Info] Caching containers")
	for _, ctr := range containers {
		config.ContainerCache[strings.Replace(ctr.Names[0], "/", "", 1)] = ctr
		//	model.Container{
		//	ID:     ctr.ID,
		//	Image:  ctr.Image,
		//	Name:   strings.Replace(ctr.Names[0], "/", "", 1),
		//	IP:     ctr.NetworkSettings.Networks[ctr.HostConfig.NetworkMode].IPAddress,
		//	Port:   ctr.Ports[0],
		//	Status: ctr.Status,
		//}
	}
	log.Println("[Info] Caching containers completed")
}

func CacheImage() {
	images, err := config.DockerClient().ImageList(context.Background(), image.ListOptions{All: true})
	if err != nil {
		log.Println("[Error]: Something wrong with get Container info ", err)
	}
	log.Println("[Info] Caching images")
	for _, img := range images {
		if len(img.RepoTags) <= 0 {
			continue
		}
		for _, tag := range img.RepoTags {
			config.ImageCache = append(config.ImageCache, tag)
		}
	}
	log.Println("[Info] Caching images completed")
}

func NewContainer() *Container {
	return &Container{}
}
