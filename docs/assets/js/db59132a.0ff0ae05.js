"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4144],{9623:(e,n,a)=>{a.r(n),a.d(n,{assets:()=>o,contentTitle:()=>l,default:()=>h,frontMatter:()=>i,metadata:()=>s,toc:()=>c});const s=JSON.parse('{"id":"quick-start","title":"Quick Start","description":"This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully.","source":"@site/versioned_docs/version-v1.1.x/quick-start.md","sourceDirName":".","slug":"/quick-start","permalink":"/eraser/docs/v1.1.x/quick-start","draft":false,"unlisted":false,"tags":[],"version":"v1.1.x","frontMatter":{"title":"Quick Start"},"sidebar":"sidebar","previous":{"title":"Installation","permalink":"/eraser/docs/v1.1.x/installation"},"next":{"title":"Architecture","permalink":"/eraser/docs/v1.1.x/architecture"}}');var t=a(4848),r=a(8453);const i={title:"Quick Start"},l=void 0,o={},c=[{value:"Deploy a DaemonSet",id:"deploy-a-daemonset",level:2},{value:"Automatically Cleaning Images",id:"automatically-cleaning-images",level:2}];function d(e){const n={a:"a",blockquote:"blockquote",code:"code",h2:"h2",p:"p",pre:"pre",...(0,r.R)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(n.p,{children:"This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully."}),"\n",(0,t.jsx)(n.h2,{id:"deploy-a-daemonset",children:"Deploy a DaemonSet"}),"\n",(0,t.jsxs)(n.p,{children:["After following the ",(0,t.jsx)(n.a,{href:"/eraser/docs/v1.1.x/installation",children:"install instructions"}),", we'll apply a demo ",(0,t.jsx)(n.code,{children:"DaemonSet"}),". For illustrative purposes, a DaemonSet is applied and deleted so the non-running images remain on all nodes. The alpine image with the ",(0,t.jsx)(n.code,{children:"3.7.3"})," tag will be used in this example. This is an image with a known critical vulnerability."]}),"\n",(0,t.jsxs)(n.p,{children:["First, apply the ",(0,t.jsx)(n.code,{children:"DaemonSet"}),":"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"cat <<EOF | kubectl apply -f -\napiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: alpine\nspec:\n  selector:\n    matchLabels:\n      app: alpine\n  template:\n    metadata:\n      labels:\n        app: alpine\n    spec:\n      containers:\n      - name: alpine\n        image: docker.io/library/alpine:3.7.3\nEOF\n"})}),"\n",(0,t.jsxs)(n.p,{children:["Next, verify that the Pods are running or completed. After the ",(0,t.jsx)(n.code,{children:"alpine"})," Pods complete, you may see a ",(0,t.jsx)(n.code,{children:"CrashLoopBackoff"})," status. This is expected behavior from the ",(0,t.jsx)(n.code,{children:"alpine"})," image and can be ignored for the tutorial."]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods\nNAME          READY   STATUS      RESTARTS     AGE\nalpine-2gh9c   1/1     Running     1 (3s ago)   6s\nalpine-hljp9   0/1     Completed   1 (3s ago)   6s\n"})}),"\n",(0,t.jsx)(n.p,{children:"Delete the DaemonSet:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl delete daemonset alpine\n"})}),"\n",(0,t.jsx)(n.p,{children:"Verify that the Pods have been deleted:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods\nNo resources found in default namespace.\n"})}),"\n",(0,t.jsxs)(n.p,{children:["To verify that the ",(0,t.jsx)(n.code,{children:"alpine"})," images are still on the nodes, exec into one of the worker nodes and list the images. If you are not using a kind cluster or Docker for your container nodes, you will need to adjust the exec command accordingly."]}),"\n",(0,t.jsx)(n.p,{children:"List the nodes:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl get nodes\nNAME                 STATUS   ROLES           AGE   VERSION\nkind-control-plane   Ready    control-plane   45m   v1.24.0\nkind-worker          Ready    <none>          45m   v1.24.0\nkind-worker2         Ready    <none>          44m   v1.24.0\n"})}),"\n",(0,t.jsxs)(n.p,{children:["List the images then filter for ",(0,t.jsx)(n.code,{children:"alpine"}),":"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ docker exec kind-worker ctr -n k8s.io images list | grep alpine\ndocker.io/library/alpine:3.7.3                                                                             application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed\ndocker.io/library/alpine@sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10           application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed\n\n"})}),"\n",(0,t.jsx)(n.h2,{id:"automatically-cleaning-images",children:"Automatically Cleaning Images"}),"\n",(0,t.jsxs)(n.p,{children:["After deploying Eraser, it will automatically clean images in a regular interval. This interval can be set using the ",(0,t.jsx)(n.code,{children:"manager.scheduling.repeatInterval"})," setting in the ",(0,t.jsx)(n.a,{href:"https://eraser-dev.github.io/eraser/docs/customization#detailed-options",children:"configmap"}),". The default interval is 24 hours (",(0,t.jsx)(n.code,{children:"24h"}),'). Valid time units are "ns", "us" (or "\xb5s"), "ms", "s", "m", "h".']}),"\n",(0,t.jsx)(n.p,{children:"Eraser will schedule eraser pods to each node in the cluster, and each pod will contain 3 containers: collector, scanner, and remover that will run to completion."}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods -n eraser-system\nNAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE\neraser-system        eraser-kind-control-plane-sb789           0/3     Completed   0                26m\neraser-system        eraser-kind-worker-j84hm                  0/3     Completed   0                26m\neraser-system        eraser-kind-worker2-4lbdr                 0/3     Completed   0                26m\neraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                26m\n"})}),"\n",(0,t.jsx)(n.p,{children:"The collector container sends the list of all images to the scanner container, which scans and reports non-compliant images to the remover container for removal of images that are non-running. Once all pods are completed, they will be automatically cleaned up."}),"\n",(0,t.jsxs)(n.blockquote,{children:["\n",(0,t.jsxs)(n.p,{children:["If you want to remove all the images periodically, you can skip the scanner container by setting the ",(0,t.jsx)(n.code,{children:"components.scanner.enabled"})," value to ",(0,t.jsx)(n.code,{children:"false"})," using the ",(0,t.jsx)(n.a,{href:"https://eraser-dev.github.io/eraser/docs/customization#detailed-options",children:"configmap"}),". In this case, each collector pod will hold 2 containers: collector and remover."]}),"\n"]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods -n eraser-system\nNAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE\neraser-system        eraser-kind-control-plane-ksk2b           0/2     Completed   0                50s\neraser-system        eraser-kind-worker-cpgqc                  0/2     Completed   0                50s\neraser-system        eraser-kind-worker2-k25df                 0/2     Completed   0                50s\neraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                55s\n"})})]})}function h(e={}){const{wrapper:n}={...(0,r.R)(),...e.components};return n?(0,t.jsx)(n,{...e,children:(0,t.jsx)(d,{...e})}):d(e)}},8453:(e,n,a)=>{a.d(n,{R:()=>i,x:()=>l});var s=a(6540);const t={},r=s.createContext(t);function i(e){const n=s.useContext(r);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function l(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(t):e.components||t:i(e.components),s.createElement(r.Provider,{value:n},e.children)}}}]);