package podforward

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"

	"github.com/b4fun/podkit/internal/logger"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/portforward"
	"k8s.io/client-go/transport/spdy"
)

type portPair struct {
	localPort  uint16
	remotePort uint16
}

func (pp portPair) Encode() string {
	return fmt.Sprintf("%d:%d", pp.localPort, pp.remotePort)
}

type portPairs []portPair

func (ps portPairs) Encode() []string {
	var rv []string
	for _, p := range ps {
		rv = append(rv, p.Encode())
	}
	return rv
}

type Forwarder struct {
	logger logger.Logger

	// restConfig is the rest config to use for the forwarder.
	restConfig *rest.Config

	// client is the kubernetes client.
	client kubernetes.Interface

	// namespace specifies the pod namespace.
	namespace string

	// labelSelector specifies the pods label selector to use.
	labelSelector string

	// ports specifies the ports to forward.
	ports portPairs
}

// Forward forwards request to pod.
func Forward(
	forwardCtx context.Context,
	restConfig *rest.Config,
	namespace string,
	options ...Option,
) (ForwardHandle, error) {
	kubeClient, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, err
	}

	forwarder := &Forwarder{
		logger: logger.NoOp,
	}
	for _, opt := range options {
		if err := opt(forwarder); err != nil {
			return nil, err
		}
	}
	if len(forwarder.ports) < 1 {
		return nil, fmt.Errorf("no ports specified")
	}

	forwarder.restConfig = restConfig
	forwarder.client = kubeClient
	forwarder.namespace = namespace
	if forwarder.logger == nil {
		forwarder.logger = logger.NoOp
	}

	return forwarder.start(forwardCtx)
}

func (fw *Forwarder) podsListOptions() metav1.ListOptions {
	return metav1.ListOptions{
		LabelSelector: fw.labelSelector,
	}
}

func (fw *Forwarder) start(forwardCtx context.Context) (ForwardHandle, error) {
	fw.logger.Log("starting forwarder")
	pods, err := fw.client.CoreV1().Pods(fw.namespace).List(forwardCtx, fw.podsListOptions())
	if err != nil {
		err = fmt.Errorf("list pods: %w", err)
		fw.logger.Log(err.Error())
		return nil, err
	}
	if len(pods.Items) < 1 {
		fw.logger.Log("no pods listed")
		return nil, fmt.Errorf("no pods found")
	}

	targetPod := pods.Items[0]
	fw.logger.Log("forwarding to pod: %s/%s (%s)", targetPod.Namespace, targetPod.Name, targetPod.UID)

	transport, upgrader, err := spdy.RoundTripperFor(fw.restConfig)
	if err != nil {
		err = fmt.Errorf("create SPDY round tripper: %w", err)
		fw.logger.Log(err.Error())
		return nil, err
	}

	stopChan := make(chan struct{}, 1)
	errChan := make(chan error, 1)
	readyChan := make(chan struct{}, 1)

	pfURL := fw.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Namespace(targetPod.Namespace).Name(targetPod.Name).
		SubResource("portforward").
		URL()

	dialer := spdy.NewDialer(upgrader, &http.Client{Transport: transport}, "POST", pfURL)
	pf, err := portforward.New(dialer, fw.ports.Encode(), stopChan, readyChan, ioutil.Discard, ioutil.Discard)
	if err != nil {
		err = fmt.Errorf("create port forwarder: %w", err)
		fw.logger.Log(err.Error())
		return nil, err
	}

	go func() {
		errChan <- pf.ForwardPorts()
	}()

	select {
	case <-forwardCtx.Done():
		// forward start timedout or cancelled
		return nil, forwardCtx.Err()
	case err := <-errChan:
		// forward failed
		fw.logger.Log(err.Error())
		return nil, err
	case <-pf.Ready:
		fw.logger.Log("port forward is ready")
		ports, err := pf.GetPorts()
		if err != nil {
			err = fmt.Errorf("get forwarded ports: %w", err)
			fw.logger.Log(err.Error())
			return nil, err
		}
		stopOnce := &sync.Once{}
		return newForwardHandle(ports, func() {
			stopOnce.Do(func() {
				fw.logger.Log("stopping port forward")
				close(stopChan)
			})
		}), nil
	}
}
