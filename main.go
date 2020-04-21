package main

import (
	"flag"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog"
)

const (
	allNamespaces = ""
)

func main() {
	var kubeconfig string
	var master string

	flag.StringVar(&kubeconfig, "kubeconfig", "/Users/deiwin/.config/k3d/k3s-default/kubeconfig.yaml", "absolute path to the kubeconfig file")
	flag.StringVar(&master, "master", "https://localhost:6443", "master url")
	flag.Parse()

	// creates the connection
	config, err := clientcmd.BuildConfigFromFlags(master, kubeconfig)
	if err != nil {
		klog.Fatal(err)
	}

	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatal(err)
	}

	podListWatcher := cache.NewListWatchFromClient(
		clientset.CoreV1().RESTClient(),
		"pods",
		allNamespaces,
		fields.Everything(),
	)
	store := cache.NewTTLStore(cache.MetaNamespaceKeyFunc, 5*time.Minute)
	reflector := cache.NewReflector(podListWatcher, &v1.Pod{}, store, 10*time.Second)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go reflector.Run(stop)

	for {
		klog.Infof("There are %d pods in the store", len(store.ListKeys()))
		time.Sleep(time.Second)
	}
}
