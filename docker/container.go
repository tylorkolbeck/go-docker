package docker

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/network"
	"github.com/moby/moby/client"
)

type Containers []ContainerStatus

type ContainerStatus struct {
	ID         string
	Name       string
	AutoRemove bool
}

type CreateOptions struct {
	Name          string
	ImageName     string
	VolumeName    string
	Env           []string
	ContainerPort string
	MountTarget   string
	HostPort      string
	AutoRemove    bool
}

type ContainerService struct {
	APIClient  *client.Client
	Containers Containers
}

func NewContainerService(apiClient *client.Client) *ContainerService {
	return &ContainerService{
		APIClient:  apiClient,
		Containers: Containers{},
	}
}

func (s *ContainerService) Create(ctx context.Context, apiClient *client.Client, options CreateOptions) (string, error) {
	reader, err := apiClient.ImagePull(ctx, options.ImageName, client.ImagePullOptions{})
	if err != nil {
		return "", fmt.Errorf("Error pulling image: %s", err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	cPort, err := network.ParsePort(options.ContainerPort)
	if err != nil {
		return "", fmt.Errorf("Error creating port: %s", err)
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
		return "", fmt.Errorf("Error creating container: %s", err)
	}

	if _, err := apiClient.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		fmt.Printf("container error: %s", err)
		return "", fmt.Errorf("Error starting container: %s", err)
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

	// logs, err := apiClient.ContainerLogs(ctx, resp.ID, client.ContainerLogsOptions{ShowStdout: true})
	// if err != nil {
	// 	panic(err)
	// }

	// stdcopy.StdCopy(os.Stdout, os.Stderr, logs)
	//
	s.Containers = append(s.Containers, ContainerStatus{
		ID:         resp.ID,
		Name:       options.Name,
		AutoRemove: options.AutoRemove,
	})

	return resp.ID, nil
}

func (s *ContainerService) StopAndRemoveAllContainers(ctx context.Context) []error {
	return StopAndRemoveContainers(ctx, s.APIClient, s.Containers)
}

func StopAndRemoveContainers(ctx context.Context, apiClient *client.Client, containers Containers) []error {
	var errors []error
	errs := StopContainers(ctx, apiClient, containers)
	if len(errs) > 0 {
		errors = append(errors, errs...)
	}

	errs = RemoveContainers(ctx, apiClient, containers)
	if len(errs) > 0 {
		errors = append(errors, errs...)
	}

	return errors
}

func StopContainers(ctx context.Context, apiClient *client.Client, containers Containers) []error {
	var errors []error
	for _, c := range containers {
		fmt.Printf("Stopping container: %s\n", c)
		_, err := apiClient.ContainerStop(ctx, c.ID, client.ContainerStopOptions{})
		if err != nil {
			errors = append(errors, err)
		}
	}

	return errors
}

func RemoveContainers(ctx context.Context, apiClient *client.Client, containers Containers) []error {
	var errors []error
	for _, c := range containers {
		if !c.AutoRemove {
			fmt.Printf("Removing container: %s\n", c)
			_, err := apiClient.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true})
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
