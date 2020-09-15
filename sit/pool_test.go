// nolint:unparam Test function parameters can be not unique
package sit_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"os"
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/Fellah/go-sit/sit/sit"
)

const (
	uniquePostfix  = "_some_unique_postfix"
	workerHostName = "worker-hostname"
	networkID      = "network-id"
)

func TestPoolStart(t *testing.T) {
	os.Setenv("HOSTNAME", workerHostName)

	tCases := map[string]struct {
		PoolFn           func(p *sit.Pool)
		RouterFn         func(rt *mux.Router)
		ErrorAssertionFn assert.ErrorAssertionFunc
		ErrorMsg         string
	}{
		"OKUntouchedEnvironment": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx:1.19.2-alpine",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx:1.19.2-alpine")(rt)
				imagePull(t, "docker.io/library/nginx", "1.19.2-alpine")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx:1.19.2-alpine", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)
			},
			ErrorAssertionFn: assert.NoError,
		},

		"OKUntouchedEnvironment_LatestImage": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx:latest",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx:latest")(rt)
				imagePull(t, "docker.io/library/nginx", "latest")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx:latest", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)
			},
			ErrorAssertionFn: assert.NoError,
		},

		"OKUntouchedEnvironment_NoTagImage": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx")(rt)
				imagePull(t, "docker.io/library/nginx", "latest")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)
			},
			ErrorAssertionFn: assert.NoError,
		},

		"OKTouchedEnvironment": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx:1.19.2-alpine",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageExist(t, "nginx:1.19.2-alpine", "image-id")(rt)
				networkExist(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx:1.19.2-alpine", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)
			},
			ErrorAssertionFn: assert.NoError,
		},

		"ErrorWorkerConainerNotExist": {
			PoolFn: func(p *sit.Pool) {
				p.SetResource("webapp", &container.Config{
					Image: "nginx:1.19.2-alpine",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx:1.19.2-alpine")(rt)
				imagePull(t, "docker.io/library/nginx", "1.19.2-alpine")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerNotExist(t, workerHostName)(rt)
			},
			ErrorAssertionFn: assert.Error,
			ErrorMsg:         "failed to determine if `worker` container",
		},
	}

	for tn, tc := range tCases {
		t.Run(tn, func(t *testing.T) {
			rt := mux.NewRouter()
			rt.NotFoundHandler = unexpectedRequest(t)
			if tc.RouterFn != nil {
				tc.RouterFn(rt)
			}

			srv := httptest.NewServer(rt)
			defer srv.Close()
			os.Setenv("DOCKER_HOST", srv.URL)

			p, err := sit.NewPool(uniquePostfix)
			require.NoError(t, err)

			if tc.PoolFn != nil {
				tc.PoolFn(p)
			}

			err = p.Start(context.Background())
			tc.ErrorAssertionFn(t, err)
			if tc.ErrorMsg != "" {
				assert.Contains(t, err.Error(), tc.ErrorMsg)
			}
		})
	}
}

func TestPoolLogsReader(t *testing.T) {
	os.Setenv("HOSTNAME", workerHostName)

	tCases := map[string]struct {
		PoolFn           func(*sit.Pool)
		RouterFn         func(*mux.Router)
		ResourceName     string
		ValueAssertionFn assert.ValueAssertionFunc
		ErrorAssertionFn assert.ErrorAssertionFunc
		ErrorMsg         string
	}{
		"OK": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx:1.19.2-alpine",
				})

				err := pool.Start(context.Background())
				require.NoError(t, err)
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx:1.19.2-alpine")(rt)
				imagePull(t, "docker.io/library/nginx", "1.19.2-alpine")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx:1.19.2-alpine", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)

				getContainerLogs(t, "webapp"+uniquePostfix)(rt)
			},
			ResourceName:     "webapp",
			ValueAssertionFn: assert.NotNil,
			ErrorAssertionFn: assert.NoError,
		},

		"ErrorFailedToFindResource": {
			PoolFn:           func(pool *sit.Pool) {},
			RouterFn:         func(rt *mux.Router) {},
			ResourceName:     "non_exist_resource",
			ValueAssertionFn: assert.Nil,
			ErrorAssertionFn: assert.Error,
			ErrorMsg:         "failed to find resource",
		},
	}

	for tn, tc := range tCases {
		t.Run(tn, func(t *testing.T) {
			rt := mux.NewRouter()
			rt.NotFoundHandler = unexpectedRequest(t)
			if tc.RouterFn != nil {
				tc.RouterFn(rt)
			}

			srv := httptest.NewServer(rt)
			defer srv.Close()
			os.Setenv("DOCKER_HOST", srv.URL)

			pool, err := sit.NewPool(uniquePostfix)
			require.NoError(t, err)

			if tc.PoolFn != nil {
				tc.PoolFn(pool)
			}

			r, err := pool.LogsReader(context.Background(), tc.ResourceName)
			tc.ErrorAssertionFn(t, err)
			if tc.ErrorMsg != "" {
				assert.Contains(t, err.Error(), tc.ErrorMsg)
			}
			tc.ValueAssertionFn(t, r)
		})
	}
}

func TestPoolStop(t *testing.T) {
	os.Setenv("HOSTNAME", workerHostName)

	tCases := map[string]struct {
		PoolFn           func(*sit.Pool)
		RouterFn         func(*mux.Router)
		ErrorAssertionFn assert.ErrorAssertionFunc
		ErrorMsg         string
	}{
		"OK": {
			PoolFn: func(pool *sit.Pool) {
				pool.SetResource("webapp", &container.Config{
					Image: "nginx:1.19.2-alpine",
				})
			},
			RouterFn: func(rt *mux.Router) {
				imageNotExist(t, "nginx:1.19.2-alpine")(rt)
				imagePull(t, "docker.io/library/nginx", "1.19.2-alpine")(rt)
				networkNotExist(t, sit.NetBaseName+uniquePostfix, "bridge")(rt)
				assertNetworkCreate(t, sit.NetBaseName+uniquePostfix, "bridge", networkID)(rt)
				containerExist(t, workerHostName)(rt)
				networkConnect(t, networkID, workerHostName)(rt)
				containerCreation(t, "nginx:1.19.2-alpine", "webapp-container-id")(rt)
				containerStart(t, "webapp-container-id")(rt)

				containerStop(t, "webapp"+uniquePostfix)(rt)
			},
			ErrorAssertionFn: assert.NoError,
		},
	}

	for tn, tc := range tCases {
		t.Run(tn, func(t *testing.T) {
			rt := mux.NewRouter()
			rt.NotFoundHandler = unexpectedRequest(t)
			if tc.RouterFn != nil {
				tc.RouterFn(rt)
			}

			srv := httptest.NewServer(rt)
			defer srv.Close()
			os.Setenv("DOCKER_HOST", srv.URL)

			pool, err := sit.NewPool(uniquePostfix)
			require.NoError(t, err)

			if tc.PoolFn != nil {
				tc.PoolFn(pool)
			}
			require.NoError(t, pool.Start(context.Background()))

			err = pool.Stop(context.Background())
			tc.ErrorAssertionFn(t, err)
			if tc.ErrorMsg != "" {
				assert.Contains(t, err.Error(), tc.ErrorMsg)
			}
		})
	}
}

func unexpectedRequest(t *testing.T) http.HandlerFunc {
	return func(_ http.ResponseWriter, r *http.Request) {
		dump, err := httputil.DumpRequest(r, true)
		if assert.NoError(t, err) {
			assert.Fail(t, "unexpected request", string(dump))
		}
	}
}

func imageNotExist(t *testing.T, ref string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/images/json", func(w http.ResponseWriter, r *http.Request) {
				var resp []types.ImageSummary
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodGet).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				args, err := filters.FromParam(r.URL.Query().Get("filters"))
				require.NoError(t, err)
				return args.Get("reference")[0] == ref
			})
	}
}

func imageExist(t *testing.T, ref, imgID string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/images/json", func(w http.ResponseWriter, r *http.Request) {
				resp := []types.ImageSummary{{ID: imgID}}
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodGet).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				args, err := filters.FromParam(r.URL.Query().Get("filters"))
				require.NoError(t, err)
				return args.Get("reference")[0] == ref
			})
	}
}

func imagePull(t *testing.T, imgName, imgTag string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/images/create", func(w http.ResponseWriter, r *http.Request) {}).
			Methods(http.MethodPost).
			Queries("fromImage", imgName).
			Queries("tag", imgTag)
	}
}

func networkNotExist(t *testing.T, netName, netDriver string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/networks", func(w http.ResponseWriter, r *http.Request) {
				var resp []types.NetworkResource
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodGet).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				args, err := filters.FromParam(r.URL.Query().Get("filters"))
				require.NoError(t, err)
				return args.Get("name")[0] == netName && args.Get("driver")[0] == netDriver
			})
	}
}

func networkExist(t *testing.T, netName, netDriver, netID string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/networks", func(w http.ResponseWriter, r *http.Request) {
				resp := []types.NetworkResource{
					{ID: networkID, Driver: "bridge"},
				}
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodGet).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				args, err := filters.FromParam(r.URL.Query().Get("filters"))
				require.NoError(t, err)
				return args.Get("name")[0] == netName && args.Get("driver")[0] == netDriver
			})
	}
}

func assertNetworkCreate(t *testing.T, netName, netDriver, netID string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/networks/create", func(w http.ResponseWriter, r *http.Request) {
				resp := types.NetworkCreateResponse{
					ID: netID,
				}
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodPost).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				var req types.NetworkCreateRequest
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				return req.Name == netName && req.Driver == netDriver
			})
	}
}

func containerExist(t *testing.T, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/containers/%s/json", ctrID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				resp := types.ContainerJSON{
					ContainerJSONBase: &types.ContainerJSONBase{ID: ctrID},
				}
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodGet)
	}
}

func containerNotExist(_ *testing.T, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/containers/%s/json", ctrID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			}).
			Methods(http.MethodGet)
	}
}

func networkConnect(t *testing.T, netID, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/networks/%s/connect", netID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {}).
			Methods(http.MethodPost).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				var req types.NetworkConnect
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				return ctrID == req.Container
			})
	}
}

func containerCreation(t *testing.T, imgName, ctrID string) func(*mux.Router) {
	return func(rt *mux.Router) {
		rt.
			HandleFunc("/{api_version}/containers/create", func(w http.ResponseWriter, r *http.Request) {
				resp := container.ContainerCreateCreatedBody{
					ID: ctrID,
				}
				require.NoError(t, json.NewEncoder(w).Encode(resp))
			}).
			Methods(http.MethodPost).
			MatcherFunc(func(r *http.Request, rm *mux.RouteMatch) bool {
				var req container.Config
				require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
				return imgName == req.Image
			})
	}
}

func containerStart(_ *testing.T, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/containers/%s/start", ctrID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {}).
			Methods(http.MethodPost)
	}
}

func containerStop(_ *testing.T, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/containers/%s/stop", ctrID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {}).
			Methods(http.MethodPost)
	}
}

func getContainerLogs(_ *testing.T, ctrID string) func(*mux.Router) {
	path := fmt.Sprintf("/{api_version}/containers/%s/logs", ctrID)

	return func(rt *mux.Router) {
		rt.
			HandleFunc(path, func(w http.ResponseWriter, r *http.Request) {}).
			Methods(http.MethodGet)
	}
}
