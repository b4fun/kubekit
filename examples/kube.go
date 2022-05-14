package examples

import (
	"flag"
	"path/filepath"

	"github.com/b4fun/kubekit/kubehelper"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/util/homedir"
)

func BindCLIFlags(fs *flag.FlagSet) *string {
	if home := homedir.HomeDir(); home != "" {
		return fs.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	}

	return fs.String("kubeconfig", "", "absolute path to the kubeconfig file")
}

func GetClusterRestConfig(kubeconfig string) (*rest.Config, error) {
	return kubehelper.LoadRestConfig("", kubeconfig)
}

func GetKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	config, err := GetClusterRestConfig(kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
