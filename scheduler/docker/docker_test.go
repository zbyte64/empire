package docker

import (
	"testing"

	"github.com/docker/engine-api/types"
	"github.com/docker/engine-api/types/swarm"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"
)

var ctx = context.Background()

func TestScheduler_Submit(t *testing.T) {
	d := new(mockDockerClient)
	s := &Scheduler{
		docker: d,
	}

	d.On("ServiceList", types.ServiceListOptions{}).Return([]swarm.Service{}, nil)

	d.On("ServiceCreate", swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "c9366591-ab68-4d49-a333-95ce5a23df68_web",
			Labels: map[string]string{
				"empire.app.id": "c9366591-ab68-4d49-a333-95ce5a23df68",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:   "remind101/acme-inc:latest",
				Command: []string{"./bin/web"},
				Env: []string{
					"RAILS_ENV=production",
				},
			},
		},
	}).Return(types.ServiceCreateResponse{}, nil)

	err := s.Submit(ctx, testApp)
	assert.NoError(t, err)

	d.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingService(t *testing.T) {
	d := new(mockDockerClient)
	s := &Scheduler{
		docker: d,
	}

	d.On("ServiceList", types.ServiceListOptions{}).Return([]swarm.Service{
		{ID: "1234", Meta: swarm.Meta{Version: swarm.Version{Index: 1}}, Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: map[string]string{"empire.app.id": testApp.ID, "empire.app.process": "web"}}}},
		{ID: "4321", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: map[string]string{"empire.app.id": testApp.ID, "empire.app.process": "worker"}}}},
	}, nil)

	d.On("ServiceUpdate", "1234", swarm.Version{Index: 2}, swarm.ServiceSpec{
		Annotations: swarm.Annotations{
			Name: "c9366591-ab68-4d49-a333-95ce5a23df68_web",
			Labels: map[string]string{
				"empire.app.id": "c9366591-ab68-4d49-a333-95ce5a23df68",
			},
		},
		TaskTemplate: swarm.TaskSpec{
			ContainerSpec: swarm.ContainerSpec{
				Image:   "remind101/acme-inc:latest",
				Command: []string{"./bin/web"},
				Env: []string{
					"RAILS_ENV=production",
				},
			},
		},
	}).Return(nil)

	d.On("ServiceRemove", "4321").Return(nil)

	err := s.Submit(ctx, testApp)
	assert.NoError(t, err)

	d.AssertExpectations(t)
}

func TestScheduler_Remove(t *testing.T) {
	d := new(mockDockerClient)
	s := &Scheduler{
		docker: d,
	}

	d.On("ServiceList", types.ServiceListOptions{}).Return([]swarm.Service{
		{ID: "1234", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: map[string]string{"empire.app.id": testApp.ID, "empire.app.process": "web"}}}},
		{ID: "4321", Spec: swarm.ServiceSpec{Annotations: swarm.Annotations{Labels: map[string]string{"empire.app.id": "foo"}}}},
	}, nil)

	d.On("ServiceRemove", "1234").Return(nil)

	err := s.Remove(ctx, testApp.ID)
	assert.NoError(t, err)

	d.AssertExpectations(t)
}

type mockDockerClient struct {
	mock.Mock
}

func (m *mockDockerClient) ServiceCreate(_ context.Context, spec swarm.ServiceSpec, headers map[string][]string) (types.ServiceCreateResponse, error) {
	args := m.Called(spec)
	return args.Get(0).(types.ServiceCreateResponse), args.Error(1)
}

func (m *mockDockerClient) ServiceList(_ context.Context, options types.ServiceListOptions) ([]swarm.Service, error) {
	args := m.Called(options)
	return args.Get(0).([]swarm.Service), args.Error(1)
}

func (m *mockDockerClient) ServiceRemove(_ context.Context, serviceID string) error {
	args := m.Called(serviceID)
	return args.Error(0)
}

func (m *mockDockerClient) ServiceUpdate(ctx context.Context, serviceID string, version swarm.Version, service swarm.ServiceSpec, headers map[string][]string) error {
	args := m.Called(serviceID, version, service)
	return args.Error(0)
}

var testApp = &scheduler.App{
	ID:   "c9366591-ab68-4d49-a333-95ce5a23df68",
	Name: "acme-inc",
	Env: map[string]string{
		"RAILS_ENV": "production",
	},
	Labels: map[string]string{
		"empire.app.id": "c9366591-ab68-4d49-a333-95ce5a23df68",
	},
	Processes: []*scheduler.Process{
		{
			Type:      "web",
			Command:   []string{"./bin/web"},
			Instances: 1,
			Image:     image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
		},
	},
}
