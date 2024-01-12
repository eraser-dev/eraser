"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[7029],{3905:(e,n,t)=>{t.d(n,{Zo:()=>p,kt:()=>u});var a=t(7294);function r(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function l(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);n&&(a=a.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,a)}return t}function i(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?l(Object(t),!0).forEach((function(n){r(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):l(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function o(e,n){if(null==e)return{};var t,a,r=function(e,n){if(null==e)return{};var t,a,r={},l=Object.keys(e);for(a=0;a<l.length;a++)t=l[a],n.indexOf(t)>=0||(r[t]=e[t]);return r}(e,n);if(Object.getOwnPropertySymbols){var l=Object.getOwnPropertySymbols(e);for(a=0;a<l.length;a++)t=l[a],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(r[t]=e[t])}return r}var s=a.createContext({}),c=function(e){var n=a.useContext(s),t=n;return e&&(t="function"==typeof e?e(n):i(i({},n),e)),t},p=function(e){var n=c(e.components);return a.createElement(s.Provider,{value:n},e.children)},d={inlineCode:"code",wrapper:function(e){var n=e.children;return a.createElement(a.Fragment,{},n)}},m=a.forwardRef((function(e,n){var t=e.components,r=e.mdxType,l=e.originalType,s=e.parentName,p=o(e,["components","mdxType","originalType","parentName"]),m=c(t),u=r,k=m["".concat(s,".").concat(u)]||m[u]||d[u]||l;return t?a.createElement(k,i(i({ref:n},p),{},{components:t})):a.createElement(k,i({ref:n},p))}));function u(e,n){var t=arguments,r=n&&n.mdxType;if("string"==typeof e||r){var l=t.length,i=new Array(l);i[0]=m;var o={};for(var s in n)hasOwnProperty.call(n,s)&&(o[s]=n[s]);o.originalType=e,o.mdxType="string"==typeof e?e:r,i[1]=o;for(var c=2;c<l;c++)i[c]=t[c];return a.createElement.apply(null,i)}return a.createElement.apply(null,t)}m.displayName="MDXCreateElement"},8611:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>s,contentTitle:()=>i,default:()=>d,frontMatter:()=>l,metadata:()=>o,toc:()=>c});var a=t(7462),r=(t(7294),t(3905));const l={title:"Quick Start"},i=void 0,o={unversionedId:"quick-start",id:"version-v1.2.x/quick-start",title:"Quick Start",description:"This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully.",source:"@site/versioned_docs/version-v1.2.x/quick-start.md",sourceDirName:".",slug:"/quick-start",permalink:"/eraser/docs/v1.2.x/quick-start",draft:!1,tags:[],version:"v1.2.x",frontMatter:{title:"Quick Start"},sidebar:"sidebar",previous:{title:"Installation",permalink:"/eraser/docs/v1.2.x/installation"},next:{title:"Architecture",permalink:"/eraser/docs/v1.2.x/architecture"}},s={},c=[{value:"Deploy a DaemonSet",id:"deploy-a-daemonset",level:2},{value:"Automatically Cleaning Images",id:"automatically-cleaning-images",level:2}],p={toc:c};function d(e){let{components:n,...t}=e;return(0,r.kt)("wrapper",(0,a.Z)({},p,t,{components:n,mdxType:"MDXLayout"}),(0,r.kt)("p",null,"This tutorial demonstrates the functionality of Eraser and validates that non-running images are removed succesfully."),(0,r.kt)("h2",{id:"deploy-a-daemonset"},"Deploy a DaemonSet"),(0,r.kt)("p",null,"After following the ",(0,r.kt)("a",{parentName:"p",href:"/eraser/docs/v1.2.x/installation"},"install instructions"),", we'll apply a demo ",(0,r.kt)("inlineCode",{parentName:"p"},"DaemonSet"),". For illustrative purposes, a DaemonSet is applied and deleted so the non-running images remain on all nodes. The alpine image with the ",(0,r.kt)("inlineCode",{parentName:"p"},"3.7.3")," tag will be used in this example. This is an image with a known critical vulnerability."),(0,r.kt)("p",null,"First, apply the ",(0,r.kt)("inlineCode",{parentName:"p"},"DaemonSet"),":"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"cat <<EOF | kubectl apply -f -\napiVersion: apps/v1\nkind: DaemonSet\nmetadata:\n  name: alpine\nspec:\n  selector:\n    matchLabels:\n      app: alpine\n  template:\n    metadata:\n      labels:\n        app: alpine\n    spec:\n      containers:\n      - name: alpine\n        image: docker.io/library/alpine:3.7.3\nEOF\n")),(0,r.kt)("p",null,"Next, verify that the Pods are running or completed. After the ",(0,r.kt)("inlineCode",{parentName:"p"},"alpine")," Pods complete, you may see a ",(0,r.kt)("inlineCode",{parentName:"p"},"CrashLoopBackoff")," status. This is expected behavior from the ",(0,r.kt)("inlineCode",{parentName:"p"},"alpine")," image and can be ignored for the tutorial."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl get pods\nNAME          READY   STATUS      RESTARTS     AGE\nalpine-2gh9c   1/1     Running     1 (3s ago)   6s\nalpine-hljp9   0/1     Completed   1 (3s ago)   6s\n")),(0,r.kt)("p",null,"Delete the DaemonSet:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl delete daemonset alpine\n")),(0,r.kt)("p",null,"Verify that the Pods have been deleted:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl get pods\nNo resources found in default namespace.\n")),(0,r.kt)("p",null,"To verify that the ",(0,r.kt)("inlineCode",{parentName:"p"},"alpine")," images are still on the nodes, exec into one of the worker nodes and list the images. If you are not using a kind cluster or Docker for your container nodes, you will need to adjust the exec command accordingly."),(0,r.kt)("p",null,"List the nodes:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl get nodes\nNAME                 STATUS   ROLES           AGE   VERSION\nkind-control-plane   Ready    control-plane   45m   v1.24.0\nkind-worker          Ready    <none>          45m   v1.24.0\nkind-worker2         Ready    <none>          44m   v1.24.0\n")),(0,r.kt)("p",null,"List the images then filter for ",(0,r.kt)("inlineCode",{parentName:"p"},"alpine"),":"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ docker exec kind-worker ctr -n k8s.io images list | grep alpine\ndocker.io/library/alpine:3.7.3                                                                             application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed\ndocker.io/library/alpine@sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10           application/vnd.docker.distribution.manifest.list.v2+json sha256:8421d9a84432575381bfabd248f1eb56f3aa21d9d7cd2511583c68c9b7511d10 2.0 MiB   linux/386,linux/amd64,linux/arm/v6,linux/arm64/v8,linux/ppc64le,linux/s390x  io.cri-containerd.image=managed\n\n")),(0,r.kt)("h2",{id:"automatically-cleaning-images"},"Automatically Cleaning Images"),(0,r.kt)("p",null,"After deploying Eraser, it will automatically clean images in a regular interval. This interval can be set using the ",(0,r.kt)("inlineCode",{parentName:"p"},"manager.scheduling.repeatInterval")," setting in the ",(0,r.kt)("a",{parentName:"p",href:"https://eraser-dev.github.io/eraser/docs/customization#detailed-options"},"configmap"),". The default interval is 24 hours (",(0,r.kt)("inlineCode",{parentName:"p"},"24h"),'). Valid time units are "ns", "us" (or "\xb5s"), "ms", "s", "m", "h".'),(0,r.kt)("p",null,"Eraser will schedule eraser pods to each node in the cluster, and each pod will contain 3 containers: collector, scanner, and remover that will run to completion."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl get pods -n eraser-system\nNAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE\neraser-system        eraser-kind-control-plane-sb789           0/3     Completed   0                26m\neraser-system        eraser-kind-worker-j84hm                  0/3     Completed   0                26m\neraser-system        eraser-kind-worker2-4lbdr                 0/3     Completed   0                26m\neraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                26m\n")),(0,r.kt)("p",null,"The collector container sends the list of all images to the scanner container, which scans and reports non-compliant images to the remover container for removal of images that are non-running. Once all pods are completed, they will be automatically cleaned up. "),(0,r.kt)("blockquote",null,(0,r.kt)("p",{parentName:"blockquote"},"If you want to remove all the images periodically, you can skip the scanner container by setting the ",(0,r.kt)("inlineCode",{parentName:"p"},"components.scanner.enabled")," value to ",(0,r.kt)("inlineCode",{parentName:"p"},"false")," using the ",(0,r.kt)("a",{parentName:"p",href:"https://eraser-dev.github.io/eraser/docs/customization#detailed-options"},"configmap"),". In this case, each collector pod will hold 2 containers: collector and remover.")),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-shell"},"$ kubectl get pods -n eraser-system\nNAMESPACE            NAME                                         READY   STATUS      RESTARTS         AGE\neraser-system        eraser-kind-control-plane-ksk2b           0/2     Completed   0                50s\neraser-system        eraser-kind-worker-cpgqc                  0/2     Completed   0                50s\neraser-system        eraser-kind-worker2-k25df                 0/2     Completed   0                50s\neraser-system        eraser-controller-manager-86cdb4cbf9-x8d7q   1/1     Running     0                55s\n")))}d.isMDXComponent=!0}}]);