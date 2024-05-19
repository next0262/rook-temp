#!/usr/bin/env -S bash -e

function start_minikube {
    if ! minikube start --force; then
        echo "Minikube start failed, trying one more time..."
        sysctl fs.protected_regular=0
        minikube start --force
    fi
}

minikube delete

./zapping_device.sh

#kubectl delete -f cluster-minikube.yaml
#kubectl delete -f nvmeof-osd.yaml
#kubectl delete -f ../examples/crds.yaml -f ../examples/common.yaml -f operator.yaml

## local version

start_minikube

eval $(minikube docker-env)
cd ../../ && make build VERSION=1.2.3 && cd -

# patch operator.yaml to use local image
sed -i -e 's|image: rook/ceph:master|image: build-f38aa6f6/ceph-amd64:latest\n          imagePullPolicy: IfNotPresent|' \
       -e 's|ROOK_LOG_LEVEL: "INFO"|ROOK_LOG_LEVEL: "DEBUG"|' ../examples/operator.yaml

kubectl apply -f ../examples/crds.yaml -f ../examples/common.yaml -f ../examples/operator.yaml
kubectl apply -f cluster-minikube.yaml
kubectl apply -f ../examples/toolbox.yaml
