#!/usr/bin/env bash

readarray -t uids < <(ps -C dockerd -o uid=)
[[ "${#uids[@]}" != "1" ]] && exit 1

if [ ${uids[0]} -ne 0 ]; then
    echo true
    exit 0
fi

echo false
exit 1
