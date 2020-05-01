package main

import (
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
	testIp = "192.0.2.1"
)

func TestProxyTargetURL(t *testing.T) {
	g := NewGomegaWithT(t)

	path := "/api/v2/trace/5af7183fb1d4cf5f"
	req := httptest.NewRequest("GET", path, nil)
	CreateDirector(CreateIndexer())(req)

	g.Expect(req.URL.String()).To(Equal("http://127.0.0.1:9410" + path))
}

func TestOwnerTagAddition(t *testing.T) {
	g := NewGomegaWithT(t)
	owner := "from_label"

	// Add requester pod to Indexer
	indexer := CreateIndexer()
	g.Expect(indexer.Add(&v1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-pod",
			Labels: map[string]string{
				"owner": owner,
			},
		},
		Status: v1.PodStatus{PodIP: testIp},
	})).To(Succeed())
	req := httptest.NewRequest("POST", "/api/v2/spans", strings.NewReader(`
	[
		{
			"id": "352bff9a74ca9ad2",
			"traceId": "5af7183fb1d4cf5f",
			"parentId": "6b221d5bc9e6496c",
			"name": "get /api",
			"timestamp": 1556604172355737,
			"duration": 1431,
			"kind": "SERVER",
			"localEndpoint": {
				"serviceName": "backend",
				"ipv4": "192.168.99.1",
				"port": 3306
			},
			"remoteEndpoint": {
				"ipv4": "172.19.0.2",
				"port": 58648
			},
			"tags": {
				"http.method": "GET",
				"http.path": "/api"
			}
		}
	]
	`))
	CreateDirector(indexer)(req)
	body, err := ioutil.ReadAll(req.Body)
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(gjson.GetBytes(body, "0.tags.owner").String()).To(Equal(owner))
}
