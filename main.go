package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"net/http/httputil"
	"os"
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

type Config struct {
	LabelTagMapping map[string]string
	ListenPort      int
}

var (
	DefaultConfig = Config{
		LabelTagMapping: map[string]string{"owner": "owner"},
		ListenPort:      9411,
	}
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

func CreateIndexer() cache.Indexer {
	return cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{ipIndex: podIpKeyFunc})
}

func getRequesterPod(indexer cache.Indexer, req *http.Request) (*v1.Pod, error) {
	clientIP, _, err := net.SplitHostPort(req.RemoteAddr)
	if err != nil {
		return &v1.Pod{}, err
	}
	podObjects, err := indexer.ByIndex(ipIndex, clientIP)
	if err != nil {
		return &v1.Pod{}, err
	}
	if klog.V(1) {
		klog.Infof("Found the following requester pod(s) for RemoteAddr \"%s\": %+v", req.RemoteAddr, podObjects)
	}
	if len(podObjects) < 1 {
		return &v1.Pod{}, fmt.Errorf("Did not find any pod objects")
	} else if len(podObjects) > 1 {
		err := fmt.Errorf("Found more than one pod object. Found %d.", len(podObjects))
		klog.Error(err)
		return &v1.Pod{}, err
	}
	pod, ok := podObjects[0].(*v1.Pod)
	if !ok {
		return &v1.Pod{}, fmt.Errorf("%+v is not a v1.Pod", podObjects[0])
	}
	return pod, nil
}

func CreateDirector(indexer cache.Indexer, cfg Config) func(req *http.Request) {
	return func(req *http.Request) {
		req.URL.Scheme = "http"
		req.URL.Host = "127.0.0.1:9410"

		if klog.V(1) {
			klog.Infof("Got request: %+v", req)
			klog.Infof("These are the pod IPs: %v", indexer.ListIndexFuncValues(ipIndex))
		}
		if req.Method != "POST" {
			if klog.V(1) {
				klog.Infof("Ignoring %s requests. Only POST requests can be modified.", req.Method)
			}
			return
		}
		if req.URL.Path != "/api/v2/spans" {
			if klog.V(1) {
				klog.Infof("Ignoring path %s. Only /api/v2/spans requests are modified.", req.URL.Path)
			}
			return
		}
		pod, err := getRequesterPod(indexer, req)
		if err != nil {
			if klog.V(1) {
				klog.Infof("Failed to find pod: %s", err)
			}
			return
		}
		tagValues := map[string]string{}
		for labelName, tagName := range cfg.LabelTagMapping {
			val := pod.ObjectMeta.Labels[labelName]
			if klog.V(1) {
				klog.Infof("Pod label %s value: \"%s\"", labelName, val)
			}
			if val == "" {
				if klog.V(1) {
					klog.Infof("Pod label %s not set", labelName)
				}
				continue
			}
			tagValues[tagName] = val
		}
		if len(tagValues) == 0 {
			if klog.V(1) {
				klog.Infof("No labels set from mapping, continuing")
			}
			return
		}
		if req.Body == nil {
			klog.Warningf("Request doesn't have a body, continuing")
			return
		}
		bodyBytes, err := ioutil.ReadAll(req.Body)
		if err != nil {
			klog.Error("Failed to read request body", err)
			// Not sure if we can/should do anything here to fail gracefully
			// without affecting the reverse proxy.
			return
		}
		// If anything fails, then restore previous body. If we do any
		// modifications then update bodyBytes in place to ensure new body is
		// used for the request
		defer func() {
			bodyBuffer := bytes.NewBuffer(bodyBytes)
			req.Body = ioutil.NopCloser(bodyBuffer)
			req.ContentLength = int64(bodyBuffer.Len())
		}()
		var spans []map[string]interface{}
		if err = json.Unmarshal(bodyBytes, &spans); err != nil {
			klog.Error("Failed to parse spans from request body", err)
			return
		}
		modified := false
		for _, span := range spans {
			tagsObj, ok := span["tags"]
			if !ok {
				if klog.V(1) {
					klog.Infof("No tags were set for span, adding one tag: %+v", span)
				}
				span["tags"] = tagValues
				modified = true
				continue
			}
			tags, ok := tagsObj.(map[string]interface{})
			if !ok {
				klog.Errorf("Couldn't parse the tags: %+v", tagsObj)
				klog.Errorf("The tags object type: %T", tagsObj)
				continue
			}
			for tagName, value := range tagValues {
				if tag, ok := tags[tagName]; ok && tag != "" {
					if klog.V(1) {
						klog.Infof("Tag %s is already set for the span, skipping: %+v", tagName, span)
					}
					continue
				}
				tags[tagName] = value
				modified = true
			}
		}
		if !modified {
			if klog.V(1) {
				klog.Infof("Didn't change any tags, continuing")
			}
			return
		}
		// Overwrite the body to be used for the request.
		bodyBytes, err = json.Marshal(spans)
		if err != nil {
			klog.Error("Failed to marshal new body", err)
			return
		}
	}
}

func ParseConfigFromEnv() (Config, error) {
	// Note that this is a shallow copy, but that shouldn't be a problem in
	// this case.
	cfg := DefaultConfig

	labelTagMappingEnv := os.Getenv("LABEL_TAG_MAPPING")
	if labelTagMappingEnv != "" {
		var labelTagMapping map[string]string
		if err := json.Unmarshal([]byte(labelTagMappingEnv), &labelTagMapping); err != nil {
			return Config{}, fmt.Errorf("Failed to parse LABEL_TAG_MAPPING env variable: %w", err)
		}
		cfg.LabelTagMapping = labelTagMapping
	}

	listenPortEnv := os.Getenv("LISTEN_PORT")
	if listenPortEnv != "" {
		var listenPort int
		if err := json.Unmarshal([]byte(listenPortEnv), &listenPort); err != nil {
			return Config{}, fmt.Errorf("Failed to parse LISTEN_PORT env variable: %w", err)
		}
		cfg.ListenPort = listenPort
	}

	return cfg, nil
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

	cfg, err := ParseConfigFromEnv()
	if err != nil {
		klog.Fatal(err)
	}
	handler := &httputil.ReverseProxy{Director: CreateDirector(indexer, cfg)}
	klog.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", cfg.ListenPort), handler))
}
