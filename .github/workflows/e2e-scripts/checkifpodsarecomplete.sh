#!/bin/bash

$(kubectl get pod --field-selector=status.phase=Succeeded -n eraser-system >> outputcompleted.txt)

numNodes=$(($(kubectl get nodes | wc -l) - 1))
numPodsCompleted=$(grep -c Completed outputcompleted.txt)
numNamePodsCompleted=$(grep -c eraser-${1} outputcompleted.txt)

if [ $numPodsCompleted -ge $numNodes ] && [ $numNamePodsCompleted -ge $numNodes ]; then
    echo "All pods completed"
    rm outputcompleted.txt
    exit 0
else
    echo "Not all pods completed"
    rm outputcompleted.txt
    exit 1
fi
