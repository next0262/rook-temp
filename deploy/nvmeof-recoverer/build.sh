#!/usr/bin/env -S bash -e
cd ../../

eval $(minikube docker-env)
make build
