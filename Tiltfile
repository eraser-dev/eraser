# Eraser Tiltfile.

print("""
-----------------------------------------------------------------
ERASER(📒 ✏️) Assumes docker installed, a kind cluster instantiated and kubectl in the right context.
-----------------------------------------------------------------
""".strip())

# build image and push into local registry.
docker_build('localhost:5001/eraser-local', context='.')

objects = read_yaml_stream('./deploy/eraser.yaml')

## point deployment yaml at local image in registry.
for o in objects:
    if o['kind'] == 'Deployment':
        o['spec']['template']['spec']['containers'][0]['args'][1]= "--eraser-image=localhost:5001/eraser-local"
        o['spec']['template']['spec']['containers'][0]['image'] = 'localhost:5001/eraser-local'

## deploy
k8s_yaml(encode_yaml_stream(objects))

# add further deployments above. Reference sample tilt file. or https://docs.tilt.dev/install.html

local_resource('Kubectl get pods',cmd='kubectl get pods -A')

## This should be a failure until you have applied imageList to your cluster.
local_resource('Kubectl get imageList',cmd='kubectl get ImageList')

local_resource('Kubectl describe imageList',cmd='kubectl describe ImageList imagelist')
