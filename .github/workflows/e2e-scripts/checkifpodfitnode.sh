#!/bin/bash

podNames="$(kubectl get pods -n eraser-system | grep eraser-${1} | awk '{print $1}')"

for podName in ${podNames}; do
    $(kubectl get events -n eraser-system --field-selector involvedObject.name="$podName" >> outputevent.txt)
    
    if grep -q "fits the node" outputevent.txt; then
        echo "$podName fit!"
        rm outputevent.txt
    else
        echo "$podName did not fit"
        rm outputevent.txt
        exit 1
    fi
done
