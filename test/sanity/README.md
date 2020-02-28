## Sanity Tests
Testing the Azure Disk CSI driver using the [`sanity`](https://github.com/kubernetes-csi/csi-test/tree/master/pkg/sanity) package test suite.

## Run Integration Tests Locally

### Prereqsuite

- A Kubernetes cluster on Azure (aks-engine / AKS)

```bash
REGISTRY=<Dockerhub registry> IMAGE_VERSION=<image version> make sanity-test
```
