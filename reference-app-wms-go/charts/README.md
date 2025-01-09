# Deploying Temporal on a Kind Cluster Using Helm

This guide walks you through creating a local [Kind](https://kind.sigs.k8s.io/) cluster, deploying [Temporal](https://temporal.io/) using [Helm](https://helm.sh/), and customizing it via `values.yaml`.

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Create Kind Cluster](#create-kind-cluster)
3. [Install Temporal on Kind](#install-temporal-on-kind)
4. [Build Docker Images ](#build-docker-images)
5. [Deploying API Server and Worker](#deploying-api-server-and-worker)

---

## Prerequisites

Make sure you have the following tools installed:

- **Docker**: Required to create and run local containers.
- **Kind**: A tool for running local Kubernetes clusters using Docker container “nodes”.
- **Helm**: A package manager for Kubernetes.

You can verify that they are installed using:

```bash
docker version
kind version
helm version
```
## Create Kind Cluster

1. Install Kind if you haven’t already. Refer to the Kind installation guide.
2. Create a new Kind cluster:
```
kind create cluster --name temporal-cluster
```
This command provisions a single-node Kind Kubernetes cluster named temporal-cluster.
3. Confirm that your Kind cluster is running:
```
kubectl cluster-info --context kind-temporal-cluster
```

## Install Temporal on Kind 

To install Temporal in a limited but working and self-contained configuration (one replica of Cassandra, Elasticsearch and each of Temporal's services, no metrics), you can run:

```
helm install \
    --repo https://go.temporal.io/helm-charts \
    --set server.replicaCount=1 \
    --set cassandra.config.cluster_size=1 \
    --set elasticsearch.replicas=1 \
    --set prometheus.enabled=false \
    --set grafana.enabled=false \
    temporaltest temporal \
    --timeout 15m
```
This configuration consumes limited resources and it is useful for small scale tests (such as using kind or minikube).

Below is an example of an environment installed in this configuration:

```
$ kubectl get pods
NAME                                           READY   STATUS    RESTARTS   AGE
temporaltest-admintools-6cdf56b869-xdxz2       1/1     Running   0          11m
temporaltest-cassandra-0                       1/1     Running   0          11m
temporaltest-frontend-5d5b6d9c59-v9g5j         1/1     Running   2          11m
temporaltest-history-64b9ddbc4b-bwk6j          1/1     Running   2          11m
temporaltest-matching-c8887ddc4-jnzg2          1/1     Running   2          11m
temporaltest-metrics-server-7fbbf65cff-rp2ks   1/1     Running   0          11m
temporaltest-web-77f68bff76-ndkzf              1/1     Running   0          11m
temporaltest-worker-7c9d68f4cf-8tzfw           1/1     Running   2          11m
```

## Build Docker Images
1. Build your Docker image(s):
```
docker build -f Dockerfile.worker -t xfieldops/worker .

docker build -f Dockerfile.server -t xfieldops/server .
```

2. Load the image into the Kind cluster
```
kind load docker-image xfieldops/worker --name temporal-cluster

kind load docker-image xfieldops/server --name temporal-cluster
```

### Deploying API Server and Worker

1. Update the ```values.yaml``` file according to your configuration
2. cd to ```charts/reference-app-wms-go``` and run:
```
helm install xfieldops .
```