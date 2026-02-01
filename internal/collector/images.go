package collector

import (
	"context"

	dtypes "github.com/docker/docker/api/types"

	"github.com/dashu-baba/docker-doctor/internal/types"
)

func collectImages(ctx context.Context, dockerHost string, apiVersion string) (*types.Images, error) {
	cli, err := newClient(dockerHost, apiVersion)
	if err != nil {
		return nil, err
	}

	images, err := cli.ImageList(ctx, dtypes.ImageListOptions{All: true})
	if err != nil {
		return nil, err
	}

	img := &types.Images{
		Count:     len(images),
		List:      make([]types.ImageInfo, 0, len(images)),
		TotalSize: 0,
	}
	for _, i := range images {
		size := uint64(i.Size)
		img.List = append(img.List, types.ImageInfo{
			ID:   i.ID,
			Size: size,
		})
		if size > 0 {
			img.TotalSize += size
		}
	}

	return img, nil
}

