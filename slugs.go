package empire

import (
	"bytes"
	"io"
	"io/ioutil"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"golang.org/x/net/context"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID          string
	Image       image.Image
	RawProcfile []byte
}

func (s *Slug) Procfile() (procfile.Procfile, error) {
	return procfile.Parse(bytes.NewReader(s.RawProcfile))
}

// SlugsCreate persists the slug.
func (s *store) SlugsCreate(slug *Slug) (*Slug, error) {
	return slugsCreate(s.db, slug)
}

// SlugsCreate inserts a Slug into the database.
func slugsCreate(db *gorm.DB, slug *Slug) (*Slug, error) {
	return slug, db.Create(slug).Error
}

// slugsService provides convenience methods for creating slugs.
type slugsService struct {
	store *store

	// extract is a function that will return an io.Reader that, when read,
	// returns the yaml Procfile. Status information can be written to the
	// io.Writer.
	extractProcfile func(image.Image, io.Writer) io.Reader
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) SlugsCreateByImage(ctx context.Context, img image.Image, out io.Writer) (*Slug, error) {
	raw, err := ioutil.ReadAll(s.extractProcfile(img, out))
	if err != nil {
		return nil, err
	}

	return s.store.SlugsCreate(&Slug{
		Image:       img,
		RawProcfile: raw,
	})
}
