package main

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"regexp"
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
	ipIndex       = "ip"
)

var (
	hostPortRegex = regexp.MustCompile(`^(.*):.*$`)
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

func getRequesterPod(indexer cache.Indexer, req *http.Request) (*v1.Pod, error) {
	match := hostPortRegex.FindStringSubmatch(req.RemoteAddr)
	if len(match) != 2 {
		return &v1.Pod{}, fmt.Errorf("RemoteAddr \"%s\" does not contain \"host:port\"", req.RemoteAddr)
	}
	host := match[1]
	podObjects, err := indexer.ByIndex(ipIndex, host)
	if err != nil {
		return &v1.Pod{}, err
	}
	if klog.V(1) {
		klog.Infof("Found the following requester pod(s) for RemoteAddr \"%s\": %+v", req.RemoteAddr, podObjects)
	}
	if len(podObjects) != 1 {
		return &v1.Pod{}, fmt.Errorf("Found %d pod objects in index instead of one", len(podObjects))
	}
	pod, ok := podObjects[0].(*v1.Pod)
	if !ok {
		return &v1.Pod{}, fmt.Errorf("%+v is not a v1.Pod", podObjects[0])
	}
	return pod, nil
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
	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{ipIndex: podIpKeyFunc})
	reflector := cache.NewReflector(podListWatcher, &v1.Pod{}, indexer, 10*time.Second)

	// Now let's start the controller
	stop := make(chan struct{})
	defer close(stop)
	go reflector.Run(stop)

	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "127.0.0.1:9410"

		klog.Infof("Got request: %+v", req)
		klog.Infof("These are the pod IPs: %v", indexer.ListIndexFuncValues(ipIndex))
		pod, err := getRequesterPod(indexer, req)
		if err != nil {
			klog.Error(err)
			return
		}
		klog.Infof("Owner: \"%s\"", pod.ObjectMeta.Labels["owner"])
	}
	handler := &httputil.ReverseProxy{Director: director}
	klog.Fatal(http.ListenAndServe(":9411", handler))
}
