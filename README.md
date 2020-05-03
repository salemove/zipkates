# Zipkin Kubernetes Metadata Sidecar

## Configuration

Env variable      | Required | Default              | Description
------------------|----------|----------------------|------------
LABEL_TAG_MAPPING | No       | `{"owner": "owner"}` | The Kubernetes Pod labels to include and the Zipkin span tag names to map them to.
LISTEN_PORT       | No       | `9411`               | The port that the proxy will listen for incoming traffic on. Defaults to the default Zipkin port.
ZIPKIN_PORT       | No       | `9410`               | The port on localhost that the proxy will send traffic to. Has to match the `QUERY_PORT` environment variable of the Zipkin container.

## Possible improvements

- [ ] Account for X-Forwarded-For header for detecting the pod IP
- [ ] Only index pods that have the specified labels
- [ ] Allow configuring the namespace of pods to index (currently indexes all namespaces)
- [ ] Check the Content-Type header before trying to parse JSON
- [ ] Support TLS termination
