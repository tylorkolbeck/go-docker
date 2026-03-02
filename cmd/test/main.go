package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go-docker/docker"
	"go-docker/server"

	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()

	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	containerService := docker.NewContainerService(apiClient)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	server := server.NewServer()
	go func() {
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server exited: %v", err)
		}
	}()

	containerOptions1 := docker.CreateOptions{
		Name:      "Postgres1",
		ImageName: "postgres:latest",
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: "postgres_data_1",
				Target: "/var/lib/postgresql",
			},
		},
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
		},
		ContainerPort: "5432/tcp",
		HostPort:      "5432",
		AutoRemove:    true,
	}

	go func() {
		fmt.Println("Setting up container 1...")
		_, err := containerService.Create(ctx, containerOptions1)
		if err != nil {
			fmt.Printf("Could not create container 1 %s", err)
			return
		}
	}()

	containerOptions2 := docker.CreateOptions{
		Name:      "Postgres2",
		ImageName: "postgres:latest",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
		},
		ContainerPort: "5432/tcp",
		HostPort:      "5433",
		AutoRemove:    true,
	}

	go func() {
		fmt.Println("Setting up container 2...")
		_, err := containerService.Create(ctx, containerOptions2)
		if err != nil {
			fmt.Printf("Could not create container 2 %s", err)
		}
	}()

	fmt.Println("System is live. Press Ctrl+C to stop.")
	<-stop

	fmt.Println("Shutdown signal received. Running Cleanup...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		fmt.Printf("Server shutdown failed:%+v", err)
	}

	fmt.Println("Shutting down...")

	errs := containerService.StopAndRemoveAllContainers(ctx)
	if len(errs) > 0 {
		for _, err := range errs {
			fmt.Printf("Stop and remove error: %s", err)
		}
	}
	fmt.Println("Shutdown complete")
}
