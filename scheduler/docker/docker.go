// Package docker implements the Scheduler interface backed by the Docker API.
// This implementation is not recommended for production use, but can be used in
// development for testing.
package docker

import (
	"errors"
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
	"github.com/remind101/empire/scheduler"
)

const (
	labelID      = "empire.app.id"
	labelProcess = "empire.app.process"
)

// dockerClient duck types the Docker client interface we use.
type dockerClient interface {
	ServiceCreate(context.Context, swarm.ServiceSpec, map[string][]string) (types.ServiceCreateResponse, error)
	ServiceList(context.Context, types.ServiceListOptions) ([]swarm.Service, error)
	ServiceRemove(context.Context, string) error
	ServiceUpdate(context.Context, string, swarm.Version, swarm.ServiceSpec, map[string][]string) error
}

// Scheduler is a scheduler.Scheduler implementation using the Services api in
// Docker.
type Scheduler struct {
	docker dockerClient
}

// New initializes a new Scheduler instance that will use the given Docker
// client to interact with Docker.
func New(c *client.Client) *Scheduler {
	return &Scheduler{
		docker: c,
	}
}

// NewEnv initializes a new Scheduler that uses a Docker client initialized from
// the DOCKER_* environment variables.
func NewEnv() (*Scheduler, error) {
	c, err := client.NewEnvClient()
	if err != nil {
		return nil, err
	}
	return New(c), nil
}

func (s *Scheduler) Submit(ctx context.Context, app *scheduler.App) error {
	services, err := s.Services(ctx, app.ID)
	if err != nil {
		return err
	}

	pMap := make(map[string]*scheduler.Process)
	for _, p := range app.Processes {
		pMap[p.Type] = p
		if service, ok := services[p.Type]; ok {
			if err := s.UpdateService(ctx, service, app, p); err != nil {
				return err
			}
		} else {
			if err := s.CreateService(ctx, app, p); err != nil {
				return err
			}
		}
	}

	for k, service := range services {
		if _, ok := pMap[k]; !ok {
			if err := s.docker.ServiceRemove(ctx, service.ID); err != nil {
				return err
			}
		}
	}

	return nil
}

func (s *Scheduler) UpdateService(ctx context.Context, service swarm.Service, app *scheduler.App, p *scheduler.Process) error {
	if err := s.docker.ServiceUpdate(ctx, service.ID, swarm.Version{Index: service.Version.Index}, swarmServiceSpec(app, p), nil); err != nil {
		return fmt.Errorf("error creating Docker service for %s process: %v", p.Type, err)
	}

	return nil
}

func (s *Scheduler) CreateService(ctx context.Context, app *scheduler.App, p *scheduler.Process) error {
	if _, err := s.docker.ServiceCreate(ctx, swarmServiceSpec(app, p), nil); err != nil {
		return fmt.Errorf("error creating Docker service for %s process: %v", p.Type, err)
	}

	return nil
}

func (s *Scheduler) Remove(ctx context.Context, app string) error {
	services, err := s.Services(ctx, app)
	if err != nil {
		return err
	}

	for _, service := range services {
		if err := s.docker.ServiceRemove(ctx, service.ID); err != nil {
			return fmt.Errorf("error removing service %s: %v", service.ID, err)
		}
	}

	return nil
}

func (s *Scheduler) Instances(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	return nil, errors.New("not implemented")
}

func (s *Scheduler) Stop(ctx context.Context, instanceID string) error {
	return errors.New("not implemented")
}

func (s *Scheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	return errors.New("not implemented")
}

// Services returns the Docker services that relate to the given application.
func (s *Scheduler) Services(ctx context.Context, app string) (map[string]swarm.Service, error) {
	services, err := s.docker.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("error fetching Docker services: %v", err)
	}

	return appServices(app, services), nil
}

// appServices filters a []swarm.Service to only include those that relate
// to the given app.
func appServices(app string, services []swarm.Service) map[string]swarm.Service {
	filtered := make(map[string]swarm.Service)

	for _, s := range services {
		labels := s.Spec.Annotations.Labels

		if labels[labelID] == app {
			filtered[labels[labelProcess]] = s
		}
	}

	return filtered
}

// swarmServiceSpec returns a swarm ServiceSpec for the given process.
func swarmServiceSpec(app *scheduler.App, p *scheduler.Process) swarm.ServiceSpec {
	name := fmt.Sprintf("%s_%s", app.ID, p.Type)

	var env []string
	for k, v := range scheduler.Env(app, p) {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	return swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: scheduler.Labels(app, p),
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:   p.Image.String(),
				Command: p.Command,
				Env:     env,
			},
		},
	}
}
