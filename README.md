# tillerc
Helm Tiller Controller

## Design Doc
https://github.com/kubernetes/helm/issues/1586#issuecomment-268055664

## Build Instructions
```sh
# dev build
./hack/make.py

# Install/Update dependency (needs glide)
glide slow

# Build Docker image
./hack/docker/setup.sh

# Push Docker image (https://hub.docker.com/r/appscode/tillerc/)
./hack/docker/setup.sh push

# Create 3rd party objects Release & ReleaseVersion (one time setup operation)
kubectl create -f ./api/extensions/helm.yaml

# Deploy to Kubernetes (one time setup operation)
kubectl run tc --image=appscode/tillerc:<tag> --replica=1

# Deploy new image
kubectl set image deployment/tc tc=appscode/tillerc:<tag>
```
