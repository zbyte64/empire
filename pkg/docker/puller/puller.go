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
	// The maximum number of times to retry.
	Max int
	// The maximum duration to wait until failing. The zero value means no
	// deadline.
	Deadline time.Duration
	// Initialal duration to wait before starting the next pull. This
	// increases exponentially. The zero value is 1 second.
	Wait time.Duration
}

// Retry wraps another puller implementation that will retry if the image is not
// found.
func Retry(p Puller, opts RetryOptions) Puller {
	if opts.Wait == 0 {
		opts.Wait = 1 * time.Second
	}

	return PullerFunc(func(ctx context.Context, img image.Image, w io.Writer) error {
		var (
			enc = json.NewEncoder(w)

			// Number of times we've retried.
			retried int
			// Last error returned
			err error
			// Amount of time to wait. Starts at 1 second -> 2 -> 4
			// -> 8 -> etc.
			wait     = opts.Wait
			deadline = time.After(opts.Deadline)
		)

		for {
			waitCh := time.After(wait)

			select {
			case <-deadline:
				if opts.Deadline != 0 {
					// Deadline reached.
					break
				}
			case <-waitCh:
				wait = wait * 2
			}

			if opts.Max != 0 && retried >= opts.Max {
				// We've retried to the maximum number of
				// retries, so return the error.
				break
			}

			err = p.Pull(ctx, img, w)

			if err != nil {
				// Add a status message to the stream.
				enc.Encode(&jsonmessage.JSONMessage{
					Status: fmt.Sprintf("%v. Retrying in %v", err.Error(), wait),
				})
			}

			retried += 1

			continue
		}

		return err
	})
}
