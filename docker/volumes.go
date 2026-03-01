package docker

import (
	"context"
	"fmt"

	"github.com/moby/moby/client"
)

func Volumes(ctx context.Context, apiClient *client.Client) {
	volumes, err := apiClient.VolumeList(ctx, client.VolumeListOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Printf("Found %d volumes: \n", len(volumes.Items))
	for _, vol := range volumes.Items {
		fmt.Printf("- Name: %s | Driver: %s | Mountpoint: %s\n", vol.Name, vol.Driver, vol.Mountpoint)
	}

	if len(volumes.Warnings) > 0 {
		fmt.Printf("Warnings: %v\n", volumes.Warnings)
	}
}
