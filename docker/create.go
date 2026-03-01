package docker

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"

	"github.com/moby/moby/api/pkg/stdcopy"
	"github.com/moby/moby/api/types/container"
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
	AutoRemove    bool
}

func Create(ctx context.Context, apiClient *client.Client, options CreateOptions) {
	reader, err := apiClient.ImagePull(ctx, options.ImageName, client.ImagePullOptions{})
	if err != nil {
		panic(err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	cPort, err := network.ParsePort(options.ContainerPort)
	if err != nil {
		panic(err)
	}

	// Create a named volume
	// vol, err := apiClient.VolumeCreate(ctx, client.VolumeCreateOptions{
	// 	Name: options.VolumeName,
	// })
	// if err != nil {
	// 	panic(err)
	// }

	containerCfg := container.Config{
		Env: options.Env,
		ExposedPorts: network.PortSet{
			cPort: struct{}{},
		},
	}

	hostCfg := container.HostConfig{
		AutoRemove: options.AutoRemove,
		// Mounts: []mount.Mount{
		// 	{
		// 		Type:   mount.TypeVolume, // or mount.TypeBind for bind mount
		// 		Source: vol.Volume.Name,
		// 		Target: options.MountTarget,
		// 	},
		// },
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
		fmt.Printf("container error: %s", err)
		panic(err)
	}

	// wait := apiClient.ContainerWait(ctx, resp.ID, client.ContainerWaitOptions{})
	// select {
	// case err := <-wait.Error:
	// 	if err != nil {
	// 		fmt.Println("SOME ERROR")
	// 		panic(err)
	// 	}
	// case <-wait.Result:
	// }

	logs, err := apiClient.ContainerLogs(ctx, resp.ID, client.ContainerLogsOptions{ShowStdout: true})
	if err != nil {
		panic(err)
	}

	stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
}
