# Zipkates: A Kubernetes Metadata Sidecar for Zipkin

Zipkates is a Kubernetes sidecar for Zipkin. It knows about the Pods in your
cluster and uses their metadata to seamlessly add tags to spans.

## Quickstart

Just change your existing Zipkin deployment as shown below and all spans sent
from pods with an `owner` label will get an `owner` tag with the label's value.

```diff
+apiVersion: rbac.authorization.k8s.io/v1beta1
+kind: ClusterRole
+metadata:
+  name: zipkin
+rules:
+- apiGroups:
+  - ""
+  resources:
+  - pods
+  - namespaces
+  verbs:
+  - list
+  - watch
+---
+apiVersion: v1
+kind: ServiceAccount
+metadata:
+  name: zipkin
+  namespace: support
+---
+apiVersion: rbac.authorization.k8s.io/v1beta1
+kind: ClusterRoleBinding
+metadata:
+  name: zipkin
+roleRef:
+  apiGroup: rbac.authorization.k8s.io
+  kind: ClusterRole
+  name: zipkin
+subjects:
+- kind: ServiceAccount
+  name: zipkin
+  namespace: support
+---
 apiVersion: apps/v1
 kind: Deployment
 metadata:
@@ -24,16 +57,33 @@ spec:
         image: openzipkin/zipkin:2.21.1
         ports:
         - name: query-port
-          containerPort: 9411
+          containerPort: 9410
         env:
           - name: STORAGE_TYPE
             value: mem
           - name: QUERY_PORT
-            value: "9411"
+            value: "9410"
         readinessProbe:
           httpGet:
             path: /health
             port: query-port
+      - name: zipkates
+        image: salemove/zipkates:v0.1.0
+        ports:
+        - name: zipkates-port
+          containerPort: 9411
+        env:
+          - name: LABEL_TAG_MAPPING
+            value: '{"owner": "owner"}'
+          - name: LISTEN_PORT
+            value: '9411'
+          - name: ZIPKIN_PORT
+            value: '9410'
+        readinessProbe:
+          httpGet:
+            path: /healthz
+            port: zipkates-port
+      serviceAccount: zipkin
```

## Configuration

Env variable      | Required | Default              | Description
------------------|----------|----------------------|------------
LABEL_TAG_MAPPING | No       | `{"owner": "owner"}` | The Kubernetes Pod labels to include and the Zipkin span tag names to map them to.
LISTEN_PORT       | No       | `9411`               | The port that the proxy will listen for incoming traffic on. Defaults to the default Zipkin port.
ZIPKIN_PORT       | No       | `9410`               | The port on localhost that the proxy will send traffic to. Has to match the `QUERY_PORT` environment variable of the Zipkin container.

## The name

- Zipkin + Kubernetes
- Zipkin + k8s
- Zipkin + k-eight-s
- Zipkin + kates
- Zipkates ¯\\\_(ツ)\_/¯

## Thank you

- [@stevesoundcloud](https://github.com/stevesoundcloud) for inspiration in
  [Using Kubernetes Pod Metadata to Improve Zipkin Traces][soundcloud-blog] and
  in private conversation.

## Known limitations

Only [v2 Zipkin API][v2-api] is supported. The v2 API [was
released][v2-release] in early 2018 and [the v1 API][v1-api] has been
deprecated since.

## Possible improvements

- [ ] Account for X-Forwarded-For header for detecting the pod IP
- [ ] Only index pods that have the specified labels
- [ ] Allow configuring the namespace of pods to index (currently indexes all namespaces)
- [ ] Check the Content-Type header before trying to parse JSON
- [ ] Support TLS termination

[soundcloud-blog]: https://developers.soundcloud.com/blog/using-kubernetes-pod-metadata-to-improve-zipkin-traces
[v1-api]: https://zipkin.io/zipkin-api/zipkin-api.yaml
[v2-api]: https://zipkin.io/zipkin-api/zipkin2-api.yaml
[v2-release]: https://github.com/openzipkin/zipkin/releases/tag/1.30.3
