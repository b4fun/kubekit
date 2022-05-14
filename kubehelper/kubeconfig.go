package kubehelper

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	clientcmdapi "k8s.io/client-go/tools/clientcmd/api"
)

// LoadRestConfig attempts to load client-go rest config automatically.
// Based on https://github.com/kubernetes-sigs/controller-runtime/blob/2f77235e25b1e42d9e4957199f3cd0f2c3fb0d72/pkg/client/config/config.go
func LoadRestConfig(
	context string,
	cliKubeConfigPath string,
) (*rest.Config, error) {
	if len(cliKubeConfigPath) > 0 {
		// specified via cli flag, use it directly
		return loadRestConfigWithContext(
			"",
			&clientcmd.ClientConfigLoadingRules{
				ExplicitPath: cliKubeConfigPath,
			},
			context,
		)
	}

	// If the recommended kubeconfig env variable is not specified,
	// try the in-cluster config.
	kubeconfigPath := os.Getenv(clientcmd.RecommendedConfigPathEnvVar)
	if len(kubeconfigPath) == 0 {
		if c, err := rest.InClusterConfig(); err == nil {
			return c, nil
		}
	}

	// If the recommended kubeconfig env variable is set, or there
	// is no in-cluster config, try the default recommended locations.
	//
	// NOTE: For default config file locations, upstream only checks
	// $HOME for the user's home directory, but we can also try
	// os/user.HomeDir when $HOME is unset.
	//
	// TODO(jlanford): could this be done upstream?
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if _, ok := os.LookupEnv("HOME"); !ok {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("could not get current user: %w", err)
		}
		loadingRules.Precedence = append(
			loadingRules.Precedence,
			filepath.Join(u.HomeDir, clientcmd.RecommendedHomeDir, clientcmd.RecommendedFileName),
		)
	}

	return loadRestConfigWithContext("", loadingRules, context)
}

func loadRestConfigWithContext(
	apiServerURL string,
	loader clientcmd.ClientConfigLoader,
	context string,
) (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		loader,
		&clientcmd.ConfigOverrides{
			ClusterInfo: clientcmdapi.Cluster{
				Server: apiServerURL,
			},
			CurrentContext: context,
		},
	).
		ClientConfig()
}
