package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/deploys"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/manager"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

// DeploysService is an interface that can be implemented to deploy images.
type DeploysService interface {
	Deploy(*images.Image) (*deploys.Deploy, error)
}

// NewDeploysService is a factory method that generates a new DeploysService.
func NewDeploysService() DeploysService {
	return &deploysService{}
}

// deploysService is a base implementation of the DeploysService
type deploysService struct {
	AppsService     *apps.Service
	ConfigsService  *configs.Service
	SlugsService    *slugs.Service
	ReleasesService *releases.Service
	ManagerService  *manager.Service
}

// Deploy deploys an Image to the platform.
func (s *deploysService) Deploy(image *images.Image) (*deploys.Deploy, error) {
	app, err := s.AppsService.FindOrCreateByRepo(image.Repo)
	if err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.ConfigsService.Head(app)

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.SlugsService.CreateByImage(image)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	release, err := s.ReleasesService.Create(app, config, slug)
	if err != nil {
		return nil, err
	}

	// Schedule the new release onto the cluster.
	if err := s.ManagerService.ScheduleRelease(release); err != nil {
		return nil, err
	}

	// We're deployed! ...
	// hopefully.
	return &deploys.Deploy{
		ID:      "1",
		Release: release,
	}, nil
}
