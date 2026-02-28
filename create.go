package main

import (
	"context"
	"io"
	"net/netip"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type CreateOptions struct {
	ImageName     string
	VolumeName    string
	Env           []string
	ContainerPort string
	MountTarget   string
	HostPort      string
}

func Create(options CreateOptions) {
	ctx := context.Background()
	apiClient, err := client.New(client.FromEnv)

	out, err := apiClient.ImagePull(ctx, options.ImageName, client.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer out.Close()
	io.Copy(os.Stdout, out)

	cPort, err := network.ParsePort(options.ContainerPort)
	if err != nil {
		panic(err)
	}

	// Create a named volume
	vol, err := apiClient.VolumeCreate(ctx, client.VolumeCreateOptions{
		Name: options.VolumeName,
	})
	if err != nil {
		panic(err)
	}

	containerCfg := container.Config{
		Env: options.Env,
		ExposedPorts: network.PortSet{
			cPort: struct{}{},
		},
	}

	hostCfg := container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume, // or mount.TypeBind for bind mount
				Source: vol.Volume.Name,
				Target: options.MountTarget,
			},
		},
		PortBindings: network.PortMap{
			cPort: []network.PortBinding{
				{
					HostIP:   netip.IPv4Unspecified(),
					HostPort: options.HostPort,
				},
			},
		},
	}

	resp, err := apiClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image:      options.ImageName,
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
}
