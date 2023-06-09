#!/bin/bash

########################
# include the magic
########################
. demo-magic.sh

# boostrap environment
pei "kind create cluster --name eraser-demo"
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
pei "docker exec eraser-demo-control-plane ctr -n k8s.io images list | grep alpine"
pei "helm install -n eraser-system eraser eraser/eraser --create-namespace"
wait
pei "kubectl get po -n eraser-system"
wait
pei "kubectl get po -n eraser-system"
pei "docker exec eraser-demo-control-plane ctr -n k8s.io images list | grep alpine"

# teardown environment
kind delete cluster --name eraser-demo