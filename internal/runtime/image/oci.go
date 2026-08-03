package image

import (
	"context"
	"fmt"
	dockerclient "github.com/moby/moby/client"
	"strings"
)

type OCIResolver struct{ client *dockerclient.Client }

func NewOCIResolver() (*OCIResolver, error) {
	client, err := dockerclient.New(dockerclient.FromEnv, dockerclient.WithAPIVersionNegotiation())
	if err != nil {
		return nil, err
	}
	return &OCIResolver{client: client}, nil
}
func (r *OCIResolver) Resolve(ctx context.Context, reference string) (string, error) {
	if strings.Contains(reference, "@sha256:") {
		return reference[strings.Index(reference, "@")+1:], nil
	}
	result, err := r.client.ImageInspect(ctx, reference)
	if err != nil {
		return "", err
	}
	for _, digest := range result.RepoDigests {
		if index := strings.Index(digest, "@sha256:"); index >= 0 {
			return digest[index+1:], nil
		}
	}
	return "", fmt.Errorf("image %s has no immutable digest", reference)
}
