package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

func StopContainers(ctx context.Context, apiClient *client.Client, containerIds []string) *[]error {
	var errors []error
	for _, id := range containerIds {
		fmt.Printf("Stopping container: %s\n", id)
		_, err := apiClient.ContainerStop(ctx, id, client.ContainerStopOptions{})
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}

func RemoveContainers(ctx context.Context, apiClient *client.Client, containerIds []string) *[]error {
	var errors []error
	for _, id := range containerIds {
		_, err := apiClient.ContainerRemove(ctx, id, client.ContainerRemoveOptions{Force: true})
		if err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return &errors
	}

	return nil
}
