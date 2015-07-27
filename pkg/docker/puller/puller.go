package puller

import (
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/docker/docker/pkg/jsonmessage"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// Puller can pull docker images.
type Puller interface {
	Pull(context.Context, image.Image, io.Writer) error
}

// PullerFunc is a function that implements the Puller interface.
type PullerFunc func(context.Context, image.Image, io.Writer) error

// Pull calls fn.
func (fn PullerFunc) Pull(ctx context.Context, img image.Image, w io.Writer) error {
	return fn(ctx, img, w)
}

// DockerPuller can pull docker images.
type DockerPuller struct {
	client *dockerutil.Client
}

// NewDockerPuller returns a new DockerPuller instance.
func NewDockerPuller(c *dockerutil.Client) *DockerPuller {
	return &DockerPuller{client: c}
}

// Pull will pull the docker image using the underlying docker client and
// writing the raw json output stream to w.
func (p *DockerPuller) Pull(ctx context.Context, img image.Image, w io.Writer) error {
	return p.client.PullImage(ctx, docker.PullImageOptions{
		Registry:      img.Registry,
		Repository:    img.Repository,
		Tag:           img.Tag,
		OutputStream:  w,
		RawJSONStream: true,
	})
}

// FakePull writes a fake jsonmessage stream to w that looks like a real docker
// pull.
func FakePull(img image.Image, w io.Writer) error {
	messages := []jsonmessage.JSONMessage{
		{Status: fmt.Sprintf("Pulling repository %s", img.Repository)},
		{Status: fmt.Sprintf("Pulling image (%s) from %s", img.Tag, img.Repository), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: fmt.Sprintf("Pulling image (%s) from %s, endpoint: https://registry-1.docker.io/v1/", img.Tag, img.Repository), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Pulling dependent layers", Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Download complete", Progress: &jsonmessage.JSONProgress{}, ID: "a1dd7097a8e8"},
		{Status: fmt.Sprintf("Status: Image is up to date for %s", img)},
	}

	enc := json.NewEncoder(w)

	for _, m := range messages {
		if err := enc.Encode(&m); err != nil {
			return err
		}
	}

	return nil
}

// FakePuller returns a Puller that writes a fake pull to w.
var FakePuller = PullerFunc(func(ctx context.Context, img image.Image, w io.Writer) error {
	return FakePull(img, w)
})

type RetryOptions struct {
	// The number of times to retry.
	Times int
	// The maximum duration to wait until failing.
	Deadline time.Duration
}

// Retry wraps another puller implementation that will retry if the image is not
// found.
func Retry(p Puller, opts RetryOptions) Puller {
	// TODO
	return p
}
