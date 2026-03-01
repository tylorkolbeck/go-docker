package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go-docker/docker"
	"go-docker/server"

	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()

	var containerIDs []string
	var mu sync.Mutex

	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := server.NewServer()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server exited: %v", err)
		}
	}()

	containerOptions1 := docker.CreateOptions{
		ImageName:  "postgres:latest",
		VolumeName: "postgres_data",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
		},
		ContainerPort: "5432/tcp",
		MountTarget:   "/var/lib/postgresql",
		HostPort:      "5432",
		AutoRemove:    true,
	}

	go func() {
		fmt.Println("Setting up container 1...")
		id, err := docker.Create(ctx, apiClient, containerOptions1)
		if err != nil {
			fmt.Printf("Could not create container 1 %s", err)
			return
		}
		mu.Lock()
		containerIDs = append(containerIDs, id)
		mu.Unlock()
	}()

	containerOptions2 := docker.CreateOptions{
		ImageName:  "postgres:latest",
		VolumeName: "postgres_data_2",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
		},
		ContainerPort: "5432/tcp",
		MountTarget:   "/var/lib/postgresql",
		HostPort:      "5433",
		AutoRemove:    true,
	}

	go func() {
		fmt.Println("Setting up container 2...")
		id, err := docker.Create(ctx, apiClient, containerOptions2)
		if err != nil {
			fmt.Printf("Could not create container 2 %s", err)
		}
		mu.Lock()
		containerIDs = append(containerIDs, id)
		mu.Unlock()
	}()

	fmt.Println("All containers created")

	fmt.Println("System is live. Press Ctrl+C to stop.")
	<-stop

	fmt.Println("Shutdown signal received. Running Cleanup...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Server shutdown failed:%+v", err)
	}

	fmt.Println("Shutting down...")

	stopAllContainers(ctx, apiClient, containerIDs)
	fmt.Println("Shutdown complete")
	// docker.Volumes(ctx, apiClient)
}

func stopAllContainers(ctx context.Context, apiClient *client.Client, containerIds []string) {
	for _, id := range containerIds {
		fmt.Printf("Stopping container: %s\n", id)
		_, err := apiClient.ContainerStop(ctx, id, client.ContainerStopOptions{})
		if err != nil {
			fmt.Printf("Could not stop container: %s. %v", id, err)
		}
		apiClient.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
		if err != nil {
			fmt.Printf("Could not remove container: %s. %v", id, err)
		}
	}
}
