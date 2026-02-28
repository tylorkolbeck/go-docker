package main

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

func main() {
	ctx := context.Background()
	apiClient, err := client.New(client.FromEnv)
	if err != nil {
		panic(err)
	}
	defer apiClient.Close()

	imageName := "postgres:latest"
	volumeName := "postgres_date"
	pgEnv := []string{
		"POSTGRES_PASSWORD=postgres",
	}
	pgCPort := "5432/tcp"
	mountTarget := "/var/lib/postgresql/data"
	hostPort := "5432"

	out, err := apiClient.ImagePull(ctx, imageName, client.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out)

	cPort, err := network.ParsePort(pgCPort)
	if err != nil {
		panic(err)
	}

	// Create a named volume
	vol, err := apiClient.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name: volumeName,
	})
	if err != nil {
		panic(err)
	}

	containerCfg := container.Config{
		Env: pgEnv,
		ExposedPorts: network.PortSet{
			cPort: struct{}{},
		},
	}

	hostCfg := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume, // or mount.TypeBind for bind mount
				Source: vol.Volume.Name,
				Target: mountTarget,
			},
		},
		PortBindings: network.PortMap{
			cPort: []network.PortBinding{
				{
					HostIP:   netip.IPv4Unspecified(),
					HostPort: hostPort,
				},
			},
		},
	}

	resp, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image:      imageName,
		Config:     &containerCfg,
		HostConfig: &hostCfg,
	},
	)
	if err != nil {
		panic(err)
	}

	if _, err := apiClient.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		panic(err)
	}

	fmt.Println(resp.ID)
}
