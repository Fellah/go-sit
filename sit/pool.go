package sit

// import (
// 	"context"
// 	"encoding/base64"
// 	"encoding/json"
// 	"io"
// 	"os"
// 	"strings"

// 	"github.com/docker/distribution/reference"
// 	"github.com/docker/docker/api/types"
// 	"github.com/docker/docker/api/types/container"
// 	"github.com/docker/docker/api/types/filters"
// 	"github.com/docker/docker/api/types/network"
// 	"github.com/docker/docker/client"
// 	"github.com/pkg/errors"
// )

// // NetBaseName base name to build unique network name for the
// // resources pool.
// const NetBaseName = "go-sit"

// Pool of resources to launch for system integration testing.
type Pool struct {
	// 	DockerClient *client.Client
	// 	Resources    []*resource

	// 	authCfgs map[string]types.AuthConfig
	// 	authStrs map[string]string

	// 	uniquePostfix string
	// 	networkID     string
	// 	rulerCtrID    string
}

// NewPool creates pool of resources.
// Pool requires unique postfix to avoid naming collisions among Docker
// containers and networks.
func NewPool(uniquePostfix string) (*Pool, error) {
	// uniquePostfix = strings.TrimSpace(uniquePostfix)
	// if uniquePostfix == "" {
	// 	return nil, errors.New("a unique postfix can't be an empty string")
	// }

	// dockerCl, err := client.NewEnvClient()
	// if err != nil {
	// 	return nil, errors.Wrap(err, "failed to initialise a new docker client")
	// }

	return &Pool{
		// DockerClient: dockerCl,

		// authCfgs: make(map[string]types.AuthConfig),
		// authStrs: make(map[string]string),

		// uniquePostfix: uniquePostfix,
	}, nil
}

// SetAuth sets authentication config to access private Docker repository.
// func (p *Pool) SetAuth(auth, user, password string) *Pool {
// 	p.authCfgs[auth] = types.AuthConfig{
// 		Username: user,
// 		Password: password,
// 		Auth:     auth,
// 	}

// 	return p
// }

// // SetResource register resource in pool to launch container for it later.
// func (p *Pool) SetResource(name string, cfg *container.Config, opts ...Option) *Pool {
// 	p.Resources = append(p.Resources, newResource(p, name, cfg, opts...))

// 	return p
// }

// // Start launches resources required for system integration testing.
// func (p *Pool) Start(ctx context.Context) error {
// 	for registry, authCfg := range p.authCfgs {
// 		encodedJSON, err := json.Marshal(authCfg)
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to marshal authentication configuration for registry %q", registry)
// 		}
// 		p.authStrs[registry] = base64.URLEncoding.EncodeToString(encodedJSON)
// 	}

// 	for _, rc := range p.Resources {
// 		if err := p.pullImage(ctx, rc.config.Image); err != nil {
// 			return errors.Wrapf(err, "failed to pull image %q", rc.config.Image)
// 		}
// 	}

// 	networkID, err := p.createNetwork(ctx)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to use network %q", p.networkName())
// 	}
// 	p.networkID = networkID

// 	hostname := os.Getenv("HOSTNAME")
// 	containerJSON, err := p.DockerClient.ContainerInspect(ctx, hostname)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to determine if `worker` container %q exists", hostname)
// 	}
// 	p.rulerCtrID = containerJSON.ID

// 	if err := p.DockerClient.NetworkConnect(ctx, p.networkID, p.rulerCtrID, &network.EndpointSettings{
// 		Aliases: []string{"worker"},
// 	}); err != nil {
// 		return errors.Wrapf(err, "failed to connect `worker` container %q to network %q", hostname, p.networkName())
// 	}

// 	for _, rc := range p.Resources {
// 		rc.networkingConfig.EndpointsConfig[p.networkName()] = &network.EndpointSettings{
// 			NetworkID: p.networkID,
// 			Aliases:   []string{rc.config.Hostname},
// 		}

// 		if err := rc.validate(); err != nil {
// 			return err
// 		}

// 		if err := rc.preLaunch(); err != nil {
// 			return err
// 		}

// 		resp, err := p.DockerClient.ContainerCreate(ctx, rc.config, rc.hostConfig, rc.networkingConfig, rc.uniqueName())
// 		if err != nil {
// 			return errors.Wrapf(err, "failed to create Docker container %q from image %q for resource", rc.uniqueName(), rc.config.Image)
// 		}
// 		rc.containerID = resp.ID

// 		if err := p.DockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
// 			return errors.Wrapf(err, "failed to start Docker container %q from image %q for resource", rc.uniqueName(), rc.config.Image)
// 		}

// 		if err := rc.postLaunch(); err != nil {
// 			return errors.Wrapf(err, "failed to execute post launch function for resource %q", rc.Name())
// 		}
// 	}

// 	return nil
// }

// // LogsReader returns container logs reader by resource name.
// func (p *Pool) LogsReader(ctx context.Context, name string) (io.ReadCloser, error) {
// 	var uniqueName string
// 	for _, rc := range p.Resources {
// 		if rc.name == name {
// 			uniqueName = rc.uniqueName()
// 			break
// 		}
// 	}

// 	if uniqueName == "" {
// 		return nil, errors.Errorf("failed to find resource %q", name)
// 	}

// 	logsOpts := types.ContainerLogsOptions{ShowStderr: true, ShowStdout: true}
// 	return p.DockerClient.ContainerLogs(ctx, uniqueName, logsOpts)
// }

// // Stop travers registered resources and stops resources' containers.
// func (p *Pool) Stop(ctx context.Context) error {
// 	for _, rc := range p.Resources {
// 		if rc.containerID == "" {
// 			continue
// 		}

// 		if err := p.DockerClient.ContainerStop(ctx, rc.uniqueName(), nil); err != nil {
// 			return err
// 		}
// 	}

// 	return nil
// }

// // Prune travers registered resources and remove resources' containers,
// // disconects `worker` container from tests network, and removes tests network.
// func (p *Pool) Prune(ctx context.Context) error {
// 	for _, rc := range p.Resources {
// 		if rc.containerID == "" {
// 			continue
// 		}

// 		rmOpts := types.ContainerRemoveOptions{RemoveVolumes: true, Force: true}
// 		if err := p.DockerClient.ContainerRemove(ctx, rc.uniqueName(), rmOpts); err != nil {
// 			return err
// 		}
// 	}

// 	if err := p.DockerClient.NetworkDisconnect(ctx, p.networkID, p.rulerCtrID, true); err != nil {
// 		return err
// 	}

// 	if err := p.DockerClient.NetworkRemove(ctx, p.networkID); err != nil {
// 		return err
// 	}

// 	return nil
// }

// // networkName builds unique name for tests network to avoid naming collision.
// func (p *Pool) networkName() string {
// 	return NetBaseName + p.uniquePostfix
// }

// func (p *Pool) pullImage(ctx context.Context, ref string) error {
// 	args := filters.NewArgs()
// 	args, err := filters.ParseFlag("reference="+ref, args)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to parse flag %q", "reference="+ref)
// 	}

// 	resp, err := p.DockerClient.ImageList(ctx, types.ImageListOptions{
// 		Filters: args,
// 	})
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to determine if image %q already exist", ref)
// 	}

// 	switch l := len(resp); {
// 	case l == 1:
// 		return nil
// 	case l > 1:
// 		return errors.New("found multiple images")
// 	}

// 	name, err := reference.ParseNormalizedNamed(ref)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to parse image reference")
// 	}

// 	tag := "latest"
// 	if tagged, ok := name.(reference.Tagged); ok {
// 		tag = tagged.Tag()
// 	}

// 	name, err = reference.WithTag(name, tag)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to combine image name and tag")
// 	}

// 	imgPullOpts := types.ImagePullOptions{}
// 	for registry, authStr := range p.authStrs {
// 		if strings.HasPrefix(name.String(), registry) {
// 			imgPullOpts.RegistryAuth = authStr
// 		}
// 	}

// 	r, err := p.DockerClient.ImagePull(ctx, name.String(), imgPullOpts)
// 	if err != nil {
// 		return errors.Wrapf(err, "failed to pull Docker image %q for resource", ref)
// 	}
// 	if _, err := io.Copy(os.Stdout, r); err != nil {
// 		return err
// 	}

// 	return nil
// }

// func (p *Pool) createNetwork(ctx context.Context) (string, error) {
// 	args := filters.NewArgs()
// 	var err error
// 	for _, flag := range []string{
// 		"name=" + p.networkName(),
// 		"driver=bridge",
// 	} {
// 		if args, err = filters.ParseFlag(flag, args); err != nil {
// 			return "", err
// 		}
// 	}

// 	netListResp, err := p.DockerClient.NetworkList(ctx, types.NetworkListOptions{
// 		Filters: args,
// 	})
// 	if err != nil {
// 		return "", err
// 	}

// 	switch l := len(netListResp); {
// 	case l == 1:
// 		return netListResp[0].ID, nil
// 	case l > 1:
// 		return "", errors.New("found multiple tests networks")
// 	}

// 	networkCreateOpts := types.NetworkCreate{
// 		CheckDuplicate: true,
// 		Driver:         "bridge",
// 	}
// 	netCreateResp, err := p.DockerClient.NetworkCreate(ctx, p.networkName(), networkCreateOpts)
// 	if err != nil {
// 		return "", err
// 	}

// 	return netCreateResp.ID, nil
// }
