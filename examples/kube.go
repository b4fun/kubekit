package examples

import (
	"flag"
	"os"
	"path/filepath"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

func BindCLIFlags(fs *flag.FlagSet) *string {
	if home := homedir.HomeDir(); home != "" {
		return fs.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	}

	return fs.String("kubeconfig", "", "absolute path to the kubeconfig file")
}

func OutOfClusterKubeClient(kubeconfig string) (kubernetes.Interface, error) {
	if kubeconfig == "" {
		if v := os.Getenv("KUBECONFIG"); v != "" {
			kubeconfig = v
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	return clientset, nil
}
