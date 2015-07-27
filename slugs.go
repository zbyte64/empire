package empire

import (
	"io"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/docker/puller"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           string
	Image        image.Image
	ProcessTypes CommandMap
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
	store     *store
	extractor Extractor
	puller    puller.Puller
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) SlugsCreateByImage(ctx context.Context, img image.Image, out io.Writer) (*Slug, error) {
	return slugsCreateByImage(ctx, s.store, s.extractor, s.puller, img, out)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func slugsCreateByImage(ctx context.Context, store *store, e Extractor, p puller.Puller, img image.Image, out io.Writer) (*Slug, error) {
	if err := p.Pull(ctx, img, out); err != nil {
		return nil, err
	}

	slug, err := slugsExtract(e, img)
	if err != nil {
		return slug, err
	}

	return store.SlugsCreate(slug)
}

// SlugsExtract extracts the process types from the image, then returns a new
// Slug instance.
func slugsExtract(e Extractor, img image.Image) (*Slug, error) {
	slug := &Slug{
		Image: img,
	}

	pt, err := e.Extract(img)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = pt

	return slug, nil
}
