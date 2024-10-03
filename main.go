package main

import (
	"droxy/config"
	"droxy/core/_http/service"
	"droxy/server"
	"log"
)

func main() {
	err := config.InitiateClientSet()
	if err != nil {
		log.Println("Error initiating client set:", err)
		return
	}
	service.CacheContainer()
	service.CacheImage()
	server.Init()
}
