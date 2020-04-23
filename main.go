package main

import (
	"fmt"
	"time"

	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"k8s.io/klog"
)

const (
	allNamespaces = ""
)

func podIpKeyFunc(obj interface{}) ([]string, error) {
	pod, ok := obj.(*v1.Pod)
	if !ok {
		return []string{}, fmt.Errorf("%v is not a v1.Pod", obj)
	}
	if pod.Status.PodIP == "" {
		return []string{}, nil
	}

	return []string{pod.Status.PodIP}, nil
}

func main() {
	config, err := rest.InClusterConfig()
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
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{"ip": podIpKeyFunc})
	reflector := cache.NewReflector(podListWatcher, &v1.Pod{}, indexer, 10*time.Second)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go reflector.Run(stop)

	for {
		klog.Infof("There are %d pods in the store", len(indexer.ListKeys()))
		klog.Infof("These are the pod keys: %v", indexer.ListKeys())
		klog.Infof("These are the pod IPs: %v", indexer.ListIndexFuncValues("ip"))
		if len(indexer.ListKeys()) > 0 {
			ip, _ := podIpKeyFunc(indexer.List()[0])
			klog.Infof("One pod has IP: %s", ip[0])
		}
		time.Sleep(time.Second)
	}
}
