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

	re := regexp.MustCompile(`^(.*):.*$`)
	director := func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "127.0.0.1:9410"

		klog.Infof("Got request: %+v", req)
		klog.Infof("These are the pod IPs: %v", indexer.ListIndexFuncValues("ip"))
		pods, err := indexer.ByIndex("ip", re.FindStringSubmatch(req.RemoteAddr)[1])
		if err != nil {
			klog.Error(err)
			return
		}
		klog.Infof("It's from this pod(s): %+v", pods)
		if len(pods) != 1 {
			klog.Errorf("%+v does not have exactly one pod", pods)
			return
		}
		podNew, ok := pods[0].(*v1.Pod)
		if !ok {
			klog.Errorf("%v is not a v1.Pod", pods)
			return
		}
		klog.Infof("Owner: \"%s\"", podNew.ObjectMeta.Labels["owner"])
	}
	handler := &httputil.ReverseProxy{Director: director}
	klog.Fatal(http.ListenAndServe(":9411", handler))
}
