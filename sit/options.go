package sit

import (
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
)

// Option is an optional configuration for resource container.
type Option func(rc *resource)

// WithHostConfig sets host configuration for resource container.
func WithHostConfig(cfg *container.HostConfig) Option {
	return func(rc *resource) {
		rc.hostConfig = cfg
	}
}

// WithNetworkingConfig sets network configuration for resource container.
func WithNetworkingConfig(cfg *network.NetworkingConfig) Option {
	return func(rc *resource) {
		rc.networkingConfig = cfg
	}
}

// WithPreLaunchFn sets function to execute before resource container start.
func WithPreLaunchFn(preFn PreLaunchFunc) Option {
	return func(rc *resource) {
		rc.preLaunchFn = preFn
	}
}

// WithPostLaunchFn sets function to execute after resource container start.
func WithPostLaunchFn(postFn PostLaunchFunc) Option {
	return func(rc *resource) {
		rc.postLaunchFn = postFn
	}
}

// WithAutoRemove sets host configuration parameter to remove resource container
// after container is stopped.
func WithAutoRemove() Option {
	return func(rc *resource) {
		rc.hostConfig.AutoRemove = true
	}
}
