# Zipkin Kubernetes Metadata Sidecar

## Configuration

Env variable      | Required | Default              | Description
------------------|----------|----------------------|------------
LABEL_TAG_MAPPING | No       | `{"owner": "owner"}` | The Kubernetes Pod labels to include and the Zipkin span tag names to map them to.
LISTEN_PORT       | No       | `9411`               | The port that the proxy will listen for incoming traffic on. Defaults to the default Zipkin port.
