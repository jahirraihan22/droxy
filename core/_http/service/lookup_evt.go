package service

import (
	"context"
	"droxy/config"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/events"
	"log"
	"time"
)

func LookUpEvent() {
	// Listen to Docker events
	messages, errs := config.DockerClient().Events(context.Background(), types.EventsOptions{})

	fmt.Println("Listening for Docker events...")

	// Handle events and errors
	for {
		select {
		case event := <-messages:
			handleDockerEvent(event)

		case err := <-errs:
			if err != nil {
				log.Println("[Warning] ", err)
			}
		}
	}
}

func handleDockerEvent(event events.Message) {
	CacheContainer()
	fmt.Printf("Action: %s | Type: %s | ID: %s | From: %s | Time: %s\n",
		event.Action, event.Type, event.ID, event.From, time.Unix(event.Time, 0).Format(time.RFC3339))
}
