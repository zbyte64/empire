package puller

import (
	"errors"
	"io"
	"io/ioutil"
	"testing"
	"time"

	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

var errTagNotFound = errors.New("Tag not found")

func TestRetry(t *testing.T) {
	var pulled int
	p := Retry(PullerFunc(func(ctx context.Context, img image.Image, w io.Writer) error {
		pulled += 1

		if pulled == 2 {
			return errTagNotFound
		}

		return nil
	}), RetryOptions{
		Max:      2,
		Deadline: 10 * time.Minute,
		Wait:     1 * time.Millisecond,
	})

	err := p.Pull(context.Background(), image.Image{}, ioutil.Discard)

	if got, want := pulled, 2; got != want {
		t.Fatalf("Got %d pulls, want %d", got, want)
	}

	if err != errTagNotFound {
		t.Fatalf("Error => %v; want %v", err, errTagNotFound)
	}
}
