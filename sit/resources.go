package sit

import (
	"strings"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/pkg/errors"
)

var errEmptyResourceName = errors.New("resource name can't be empty")

// PreLaunchFunc is function to execute before resource container start.
type PreLaunchFunc func(rc Resource) error

// PostLaunchFunc is function to execute after resource container start.
type PostLaunchFunc func(rc Resource) error

// Resource is a set of configurations for resource container.
type Resource interface {
	Name() string
	Config() *container.Config
	HostConfig() *container.HostConfig
	NetworkingConfig() *network.NetworkingConfig
}

type resource struct {
	name             string
	config           *container.Config
	hostConfig       *container.HostConfig
	networkingConfig *network.NetworkingConfig

	pool *Pool

	preLaunchFn  PreLaunchFunc
	postLaunchFn PostLaunchFunc

	containerID string
}

func newResource(pool *Pool, name string, cfg *container.Config, opts ...Option) *resource {
	name = strings.TrimSpace(name)

	cfg.Hostname = strings.TrimSpace(cfg.Hostname)
	if cfg.Hostname == "" {
		cfg.Hostname = name
	}

	rc := &resource{
		name:       name,
		config:     cfg,
		hostConfig: &container.HostConfig{},
		networkingConfig: &network.NetworkingConfig{
			EndpointsConfig: make(map[string]*network.EndpointSettings),
		},

		pool: pool,
	}

	for _, opt := range opts {
		opt(rc)
	}

	return rc
}

// Name provides resource name.
func (rc *resource) Name() string {
	return rc.name
}

// Config is getter for container configureation.
func (rc *resource) Config() *container.Config {
	return rc.config
}

// Config is getter for container host configureation.
func (rc *resource) HostConfig() *container.HostConfig {
	return rc.hostConfig
}

// Config is getter for container network configureation.
func (rc *resource) NetworkingConfig() *network.NetworkingConfig {
	return rc.networkingConfig
}

// Validate resource parameters.
func (rc *resource) validate() error {
	if rc.name == "" {
		return errEmptyResourceName
	}
	return nil
}

// UniqName provides resource unique name builded on resource name and
// resources pool's unique postfix.
func (rc *resource) uniqueName() string {
	return rc.name + rc.pool.uniquePostfix
}

func (rc *resource) preLaunch() error {
	if rc.preLaunchFn != nil {
		return rc.preLaunchFn(rc)
	}

	return nil
}

func (rc *resource) postLaunch() error {
	if rc.postLaunchFn != nil {
		return rc.postLaunchFn(rc)
	}

	return nil
}
