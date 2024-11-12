"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[7466],{1368:(e,n,s)=>{s.r(n),s.d(n,{assets:()=>o,contentTitle:()=>i,default:()=>m,frontMatter:()=>l,metadata:()=>r,toc:()=>c});const r=JSON.parse('{"id":"manual-removal","title":"Manual Removal","description":"Create an ImageList and specify the images you would like to remove. In this case, the image docker.io/library/alpine:3.7.3 will be removed.","source":"@site/docs/manual-removal.md","sourceDirName":".","slug":"/manual-removal","permalink":"/eraser/docs/next/manual-removal","draft":false,"unlisted":false,"tags":[],"version":"current","frontMatter":{"title":"Manual Removal"},"sidebar":"sidebar","previous":{"title":"Architecture","permalink":"/eraser/docs/next/architecture"},"next":{"title":"Exclusion","permalink":"/eraser/docs/next/exclusion"}}');var a=s(4848),t=s(8453);const l={title:"Manual Removal"},i=void 0,o={},c=[];function d(e){const n={blockquote:"blockquote",code:"code",p:"p",pre:"pre",...(0,t.R)(),...e.components};return(0,a.jsxs)(a.Fragment,{children:[(0,a.jsxs)(n.p,{children:["Create an ",(0,a.jsx)(n.code,{children:"ImageList"})," and specify the images you would like to remove. In this case, the image ",(0,a.jsx)(n.code,{children:"docker.io/library/alpine:3.7.3"})," will be removed."]}),"\n",(0,a.jsx)(n.pre,{children:(0,a.jsx)(n.code,{className:"language-shell",children:'cat <<EOF | kubectl apply -f -\napiVersion: eraser.sh/v1alpha1\nkind: ImageList\nmetadata:\n  name: imagelist\nspec:\n  images:\n    - docker.io/library/alpine:3.7.3   # use "*" for all non-running images\nEOF\n'})}),"\n",(0,a.jsxs)(n.blockquote,{children:["\n",(0,a.jsxs)(n.p,{children:[(0,a.jsx)(n.code,{children:"ImageList"})," is a cluster-scoped resource and must be called imagelist. ",(0,a.jsx)(n.code,{children:'"*"'})," can be specified to remove all non-running images instead of individual images."]}),"\n"]}),"\n",(0,a.jsxs)(n.p,{children:["Creating an ",(0,a.jsx)(n.code,{children:"ImageList"})," should trigger an ",(0,a.jsx)(n.code,{children:"ImageJob"})," that will deploy Eraser pods on every node to perform the removal given the list of images."]}),"\n",(0,a.jsx)(n.pre,{children:(0,a.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods -n eraser-system\neraser-system        eraser-controller-manager-55d54c4fb6-dcglq   1/1     Running   0          9m8s\neraser-system        eraser-kind-control-plane                    1/1     Running   0          11s\neraser-system        eraser-kind-worker                           1/1     Running   0          11s\neraser-system        eraser-kind-worker2                          1/1     Running   0          11s\n"})}),"\n",(0,a.jsx)(n.p,{children:"Pods will run to completion and the images will be removed."}),"\n",(0,a.jsx)(n.pre,{children:(0,a.jsx)(n.code,{className:"language-shell",children:"$ kubectl get pods -n eraser-system\neraser-system        eraser-controller-manager-6d6d5594d4-phl2q   1/1     Running     0          4m16s\neraser-system        eraser-kind-control-plane                    0/1     Completed   0          22s\neraser-system        eraser-kind-worker                           0/1     Completed   0          22s\neraser-system        eraser-kind-worker2                          0/1     Completed   0          22s\n"})}),"\n",(0,a.jsxs)(n.p,{children:["The ",(0,a.jsx)(n.code,{children:"ImageList"})," custom resource status field will contain the status of the last job. The success and failure counts indicate the number of nodes the Eraser agent was run on."]}),"\n",(0,a.jsx)(n.pre,{children:(0,a.jsx)(n.code,{className:"language-shell",children:"$ kubectl describe ImageList imagelist\n...\nStatus:\n  Failed:     0\n  Success:    3\n  Timestamp:  2022-02-25T23:41:55Z\n...\n"})}),"\n",(0,a.jsx)(n.p,{children:"Verify the unused images are removed."}),"\n",(0,a.jsx)(n.pre,{children:(0,a.jsx)(n.code,{className:"language-shell",children:"$ docker exec kind-worker ctr -n k8s.io images list | grep alpine\n"})}),"\n",(0,a.jsx)(n.p,{children:"If the image has been successfully removed, there will be no output."})]})}function m(e={}){const{wrapper:n}={...(0,t.R)(),...e.components};return n?(0,a.jsx)(n,{...e,children:(0,a.jsx)(d,{...e})}):d(e)}},8453:(e,n,s)=>{s.d(n,{R:()=>l,x:()=>i});var r=s(6540);const a={},t=r.createContext(a);function l(e){const n=r.useContext(t);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function i(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(a):e.components||a:l(e.components),r.createElement(t.Provider,{value:n},e.children)}}}]);