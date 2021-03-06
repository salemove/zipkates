package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// testIp is the hardcoded value used by httptest.NewRequest in RemoteAddr
	testIp      = "192.0.2.1"
	differentIp = "10.0.0.1"
)

func TestProxyTargetURL(t *testing.T) {
	g := NewWithT(t)

	path := "/api/v2/trace/5af7183fb1d4cf5f"
	req := httptest.NewRequest("GET", path, nil)
	CreateDirector(CreateIndexer(), DefaultConfig)(req)

	g.Expect(req.URL.String()).To(Equal("http://127.0.0.1:9410" + path))
}

func TestDifferentZipkinPort(t *testing.T) {
	g := NewWithT(t)

	path := "/api/v2/trace/5af7183fb1d4cf5f"
	req := httptest.NewRequest("GET", path, nil)
	cfg := DefaultConfig
	cfg.ZipkinPort = 8080
	CreateDirector(CreateIndexer(), cfg)(req)

	g.Expect(req.URL.String()).To(Equal("http://127.0.0.1:8080" + path))
}

func TestOwnerTagAddition(t *testing.T) {
	g := NewWithT(t)
	owner := "from_label"

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": owner}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
		}))),
	)
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.owner").String()).To(Equal(owner))
}

func TestKeepOriginalOwnerTag(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": "from_label"}))).To(Succeed())

	fromSpan := "from_span"
	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
			"owner":       fromSpan,
		}))),
	)
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.owner").String()).To(Equal(fromSpan))
}

func TestEmptyTags(t *testing.T) {
	g := NewWithT(t)
	owner := "from_label"

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": owner}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{}))),
	)
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.owner").String()).To(Equal(owner))
}

func TestMissingTags(t *testing.T) {
	g := NewWithT(t)
	owner := "from_label"

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": owner}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, nil))),
	)
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.owner").String()).To(Equal(owner))
}

func TestZeroSpans(t *testing.T) {
	g := NewWithT(t)

	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader("[]"))
	CreateDirector(CreateIndexer(), DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal("[]"))
}

func TestMultipleSpans(t *testing.T) {
	g := NewWithT(t)
	fromLabel := "from_label"
	fromSpan := "from_span"

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": fromLabel}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s, %s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
			"owner":       fromSpan,
		}), span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
		}))),
	)
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "#.tags.owner").String()).
		To(Equal(fmt.Sprintf(`["%s","%s"]`, fromSpan, fromLabel)))
}

func TestDifferentPodIP(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", differentIp, map[string]string{"owner": "from_label"}))).To(Succeed())

	originalBody := fmt.Sprintf("[%s]", span(g, map[string]string{
		"http.method": "GET",
		"http.path":   "/api",
	}))
	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader(originalBody))
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal(originalBody))
}

func TestPodWithoutOwnerLabel(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": ""}))).To(Succeed())

	originalBody := fmt.Sprintf("[%s]", span(g, map[string]string{
		"http.method": "GET",
		"http.path":   "/api",
	}))
	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader(originalBody))
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal(originalBody))
}

func TestSpansNotAnArray(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": "from_label"}))).To(Succeed())

	originalBody := span(g, map[string]string{
		"http.method": "GET",
		"http.path":   "/api",
	})
	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader(originalBody))
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal(originalBody))
}

func TestDifferentPath(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": "from_label"}))).To(Succeed())

	originalBody := fmt.Sprintf("[%s]", span(g, map[string]string{
		"http.method": "GET",
		"http.path":   "/api",
	}))
	req := httptest.NewRequest("POST", "/api/v1/spans", strings.NewReader(originalBody))
	CreateDirector(indexer, DefaultConfig)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal(originalBody))
}

func TestDifferentTagAddition(t *testing.T) {
	g := NewWithT(t)
	labelName := "other_label"
	tagName := "other_tag"
	labelValue := "from_label"

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{labelName: labelValue}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
		}))),
	)
	cfg := DefaultConfig
	cfg.LabelTagMapping = map[string]string{labelName: tagName}
	CreateDirector(indexer, cfg)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags."+tagName).String()).To(Equal(labelValue))
}

func TestMultipleTagAddition(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{
		"label_a": "label_a",
		"label_b": "label_b",
	}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
		}))),
	)
	cfg := DefaultConfig
	cfg.LabelTagMapping = map[string]string{
		"label_a": "tag_a",
		"label_b": "tag_b",
	}
	CreateDirector(indexer, cfg)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.tag_a").String()).To(Equal("label_a"))
	g.Expect(gjson.GetBytes(body, "0.tags.tag_b").String()).To(Equal("label_b"))
}

func TestPartialMultipleTagAddition(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{
		"label_a": "label_a",
		"label_b": "label_b",
	}))).To(Succeed())

	req := httptest.NewRequest(
		"POST", "/api/v2/spans",
		strings.NewReader(fmt.Sprintf("[%s]", span(g, map[string]string{
			"http.method": "GET",
			"http.path":   "/api",
			"tag_a":       "tag_a",
		}))),
	)
	cfg := DefaultConfig
	cfg.LabelTagMapping = map[string]string{
		"label_a": "tag_a",
		"label_b": "tag_b",
	}
	CreateDirector(indexer, cfg)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(gjson.GetBytes(body, "0.tags.tag_a").String()).To(Equal("tag_a"))
	g.Expect(gjson.GetBytes(body, "0.tags.tag_b").String()).To(Equal("label_b"))
}

func TestEmptyMapping(t *testing.T) {
	g := NewWithT(t)

	indexer := CreateIndexer()
	g.Expect(indexer.Add(pod("test-pod", testIp, map[string]string{"owner": "from_label"}))).To(Succeed())

	originalBody := fmt.Sprintf("[%s]", span(g, map[string]string{
		"http.method": "GET",
		"http.path":   "/api",
	}))
	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader(originalBody))
	cfg := DefaultConfig
	cfg.LabelTagMapping = map[string]string{}
	CreateDirector(indexer, cfg)(req)

	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(body)).To(Equal(originalBody))
}

func pod(name, ip string, labels map[string]string) *v1.Pod {
	return &v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: v1.PodStatus{PodIP: ip},
	}
}

func span(g *WithT, tags map[string]string) string {
	span := map[string]interface{}{
		"id":        "352bff9a74ca9ad2",
		"traceId":   "5af7183fb1d4cf5f",
		"parentId":  "6b221d5bc9e6496c",
		"name":      "get /api",
		"timestamp": 1556604172355737,
		"duration":  1431,
		"kind":      "SERVER",
		"localEndpoint": map[string]interface{}{
			"serviceName": "backend",
			"ipv4":        "192.168.99.1",
			"port":        3306,
		},
		"remoteEndpoint": map[string]interface{}{
			"ipv4": "172.19.0.2",
			"port": 58648,
		},
	}
	if tags != nil {
		span["tags"] = tags
	}
	result, err := json.Marshal(span)
	g.Expect(err).NotTo(HaveOccurred())
	return string(result)
}
