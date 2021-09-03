#!/bin/bash

nodeNames="$(kubectl get nodes -n eraser-system | grep ${1} | awk '{print $1}')"


for nodeName in ${nodeNames}; do
    $(docker exec "$nodeName" ctr -n k8s.io images list >> outputimagelist.txt)

    if grep -q docker.io/library/hello-world:latest outputimagelist.txt; then
        echo "Node $nodeName: Test Image was not removed"
        rm outputimagelist.txt
        exit 1
    else
        echo "Node $nodeName: Test Image was removed"
        rm outputimagelist.txt
    fi
done
