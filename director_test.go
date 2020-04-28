package main

import (
	"net/http/httptest"
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/client-go/tools/cache"
)

func TestProxyTargetURL(t *testing.T) {
	g := NewGomegaWithT(t)

	indexer := cache.NewIndexer(cache.MetaNamespaceKeyFunc, cache.Indexers{ipIndex: podIpKeyFunc})
	path := "/api/v2/trace/5af7183fb1d4cf5f"
	req := httptest.NewRequest("GET", path, nil)
	director(indexer)(req)

	g.Expect(req.URL.String()).To(Equal("http://127.0.0.1:9410" + path))
}