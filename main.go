package main

import (
	"context"
	"fmt"
	"sync"

	"go-docker/docker"

	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()
	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		panic(err)
	}

	defer apiClient.Close()

	var wg sync.WaitGroup

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

	wg.Go(func() {
		docker.Create(ctx, apiClient, containerOptions1)
	})

	containerOptions2 := docker.CreateOptions{
		ImageName:  "postgres:latest",
		VolumeName: "postgres_data_2",
		Env: []string{
			"POSTGRES_PASSWORD=postgres",
		},
		ContainerPort: "5432/tcp",
		MountTarget:   "/var/lib/postgresql",
		HostPort:      "5433",
		AutoRemove:    false,
	}

	wg.Go(func() {
		docker.Create(ctx, apiClient, containerOptions2)
	})
	wg.Wait()

	docker.Volumes(ctx, apiClient)
	fmt.Println("All container started")
}
