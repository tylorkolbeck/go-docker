package docker

import (
	"context"
	"fmt"
	"io"
	"net/netip"
	"os"
	"sync"

	"github.com/moby/moby/api/types/container"
	"github.com/moby/moby/api/types/mount"
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
	Mounts        []mount.Mount
	ImageName     string
	Env           []string
	ContainerPort string
	HostPort      string
	AutoRemove    bool
}

type ContainerService struct {
	APIClient  *client.Client
	Containers Containers
	mu         sync.Mutex
}

func NewContainerService(apiClient *client.Client) *ContainerService {
	return &ContainerService{
		APIClient:  apiClient,
		Containers: Containers{},
	}
}

func (s *ContainerService) Create(ctx context.Context, options CreateOptions) (string, error) {
	reader, err := s.APIClient.ImagePull(ctx, options.ImageName, client.ImagePullOptions{})
	if err != nil {
		return "", fmt.Errorf("error pulling image: %s", err)
	}
	defer reader.Close()
	io.Copy(os.Stdout, reader)

	cPort, err := network.ParsePort(options.ContainerPort)
	if err != nil {
		return "", fmt.Errorf("error creating port: %s", err)
	}

	containerCfg := container.Config{
		Env: options.Env,
		ExposedPorts: network.PortSet{
			cPort: struct{}{},
		},
	}

	hostCfg := container.HostConfig{
		AutoRemove: options.AutoRemove,
		Mounts:     options.Mounts,
		PortBindings: network.PortMap{
			cPort: []network.PortBinding{
				{
					HostIP:   netip.IPv4Unspecified(),
					HostPort: options.HostPort,
				},
			},
		},
	}

	resp, err := s.APIClient.ContainerCreate(ctx, client.ContainerCreateOptions{
		Image:      options.ImageName,
		Config:     &containerCfg,
		HostConfig: &hostCfg,
	},
	)
	if err != nil {
		return "", fmt.Errorf("error creating container: %s", err)
	}

	if _, err := s.APIClient.ContainerStart(ctx, resp.ID, client.ContainerStartOptions{}); err != nil {
		fmt.Printf("container error: %s", err)
		return "", fmt.Errorf("error starting container: %s", err)
	}

	s.mu.Lock()
	s.Containers = append(s.Containers, ContainerStatus{
		ID:         resp.ID,
		Name:       options.Name,
		AutoRemove: options.AutoRemove,
	})
	s.mu.Unlock()

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
		fmt.Printf("Stopping container: %+v\n", c)
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
			fmt.Printf("Removing container: %+v\n", c)
			_, err := apiClient.ContainerRemove(ctx, c.ID, client.ContainerRemoveOptions{Force: true})
			if err != nil {
				errors = append(errors, err)
			}
		}
	}

	return errors
}
