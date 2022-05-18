# Eraser Tiltfile.

print("""
-----------------------------------------------------------------
ERASER(📒 ✏️) Assumes docker installed, a kind cluster instantiated and kubectl in the right context.
-----------------------------------------------------------------
""".strip())

docker_build('eraser-build', context='.')

k8s_yaml('./deploy/eraser.yaml')
# add further deployments above. Reference sample tilt file. or https://docs.tilt.dev/install.html

local_resource('Kubectl get pods',cmd='kubectl get pods -A')
local_resource('Kubectl get imageList',cmd='kubectl get ImageList -A')
local_resource('Kubectl describe imageList',cmd='kubectl describe ImageList imagelist -A')

# Extensions are open-source, pre-packaged functions that extend Tilt
#
#   More info: https://github.com/tilt-dev/tilt-extensions

load('ext://git_resource', 'git_checkout')
