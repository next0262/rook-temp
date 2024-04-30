#!/usr/bin/env -S bash -e

#minikube delete

kubectl delete -f cluster-minikube.yaml
kubectl delete -f nvmeof-osd.yaml
kubectl delete -f ../examples/crds.yaml -f ../examples/common.yaml -f operator.yaml

## local version
# minikube start --force

# patch operator.yaml to use local image
sed -i -e 's|image: rook/ceph:master|image: build-90b200cf/ceph-amd64:latest\n          imagePullPolicy: IfNotPresent|' \
       -e 's|ROOK_LOG_LEVEL: "INFO"|ROOK_LOG_LEVEL: "DEBUG"|' ../examples/operator.yaml

kubectl create -f ../examples/crds.yaml -f ../examples/common.yaml -f operator.yaml
# kubectl apply -f nvmeof-osd.yaml
# kubectl apply -f cluster-minikube.yaml
