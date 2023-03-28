#!/bin/bash

########################
# include the magic
########################
. demo-magic.sh

# boostrap environment
pei "kind create cluster"
sleep 10
pei "kubectl apply -f ds.yaml"
sleep 10
clear

# demo commands
pei "kubectl get pods"
sleep 10
pei "kubectl delete daemonset alpine"
sleep 5
pei "kubectl get pods"
pei "docker exec kind-control-plane ctr -n k8s.io images list | grep alpine"
pei "helm install -n eraser-system eraser ../manifest_staging/charts/eraser --create-namespace"
sleep 5
pei "kubectl get po -n eraser-system"
sleep 60
pei "kubectl get po -n eraser-system"
pei "docker exec kind-control-plane ctr -n k8s.io images list | grep alpine"

# teardown environment
kind delete cluster