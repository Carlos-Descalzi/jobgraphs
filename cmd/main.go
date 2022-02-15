package main

import (
	"flag"
	"fmt"

	ifces "ced.io/jobgraphs/client/v1alpha1"
	controller "ced.io/jobgraphs/pkg/controller"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"knative.dev/pkg/signals"
)

var (
	masterURL         string
	kubeconfig        string
	logLevel          string
	inputWorkerCount  int
	outputWorkerCount int
)

func init() {
	flag.StringVar(&kubeconfig, "kubeconfig", "", "Path to kube config")
	flag.StringVar(&masterURL, "masterURL", "", "Kubernetes API Server")
	flag.StringVar(&logLevel, "log-level", "info", "Log level")
	flag.IntVar(&inputWorkerCount, "input-worker-count", 1, "Number of input workers")
	flag.IntVar(&outputWorkerCount, "output-worker-count", 1, "Number of output workers")
}

func main() {
	fmt.Printf("Starting job graph controller\n")
	klog.InitFlags(nil)
	flag.Set("logtostderr", "true")
	flag.Parse()

	logger, err := setupLogger(logLevel)

	defer logger.Sync()

	cfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeconfig)

	if err != nil {
		logger.Panic(err)
	}

	httpClient, err := rest.HTTPClientFor(cfg)

	if err != nil {
		logger.Panic(err)
	}

	ifces.AddToScheme(scheme.Scheme)

	client, err := kubernetes.NewForConfigAndClient(cfg, httpClient)

	if err != nil {
		logger.Panic(err)
	}

	jobGraphIfce, err := ifces.JobGraphInterfaceNew(cfg, httpClient)

	if err != nil {
		logger.Panic(err)
	}

	ctrl, err := controller.JobGraphsControllerNew(inputWorkerCount, outputWorkerCount, client, jobGraphIfce, logger)

	if err != nil {
		logger.Panic(err)
	}

	stopCh := signals.SetupSignalHandler()

	ctrl.Start()
	<-stopCh
	ctrl.Stop()
	logger.Info("Stopping job graph controller")
}
