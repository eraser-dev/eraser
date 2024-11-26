"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[8070],{7208:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>l,contentTitle:()=>o,default:()=>d,frontMatter:()=>a,metadata:()=>r,toc:()=>c});const r=JSON.parse('{"id":"faq","title":"FAQ","description":"Why am I still seeing vulnerable images?","source":"@site/docs/faq.md","sourceDirName":".","slug":"/faq","permalink":"/eraser/docs/next/faq","draft":false,"unlisted":false,"tags":[],"version":"current","frontMatter":{"title":"FAQ"},"sidebar":"sidebar","previous":{"title":"Trivy","permalink":"/eraser/docs/next/trivy"},"next":{"title":"Contributing","permalink":"/eraser/docs/next/contributing"}}');var i=t(4848),s=t(8453);const a={title:"FAQ"},o=void 0,l={},c=[{value:"Why am I still seeing vulnerable images?",id:"why-am-i-still-seeing-vulnerable-images",level:2},{value:"How is Eraser different from Kubernetes garbage collection?",id:"how-is-eraser-different-from-kubernetes-garbage-collection",level:2}];function u(e){const n={a:"a",code:"code",h2:"h2",li:"li",p:"p",strong:"strong",ul:"ul",...(0,s.R)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.h2,{id:"why-am-i-still-seeing-vulnerable-images",children:"Why am I still seeing vulnerable images?"}),"\n",(0,i.jsxs)(n.p,{children:["Eraser currently targets ",(0,i.jsx)(n.strong,{children:"non-running"})," images, so any vulnerable images that are currently running will not be removed. In addition, the default vulnerability scanning with Trivy removes images with ",(0,i.jsx)(n.code,{children:"CRITICAL"})," vulnerabilities. Any images with lower vulnerabilities will not be removed. This can be configured using the ",(0,i.jsx)(n.a,{href:"https://eraser-dev.github.io/eraser/docs/customization#scanner-options",children:"configmap"}),"."]}),"\n",(0,i.jsx)(n.h2,{id:"how-is-eraser-different-from-kubernetes-garbage-collection",children:"How is Eraser different from Kubernetes garbage collection?"}),"\n",(0,i.jsxs)(n.p,{children:["The native garbage collection in Kubernetes works a bit differently than Eraser. By default, garbage collection begins when disk usage reaches 85%, and stops when it gets down to 80%. More details about Kubernetes garbage collection can be found in the ",(0,i.jsx)(n.a,{href:"https://kubernetes.io/docs/concepts/architecture/garbage-collection/",children:"Kubernetes documentation"}),", and configuration options can be found in the ",(0,i.jsx)(n.a,{href:"https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/",children:"Kubelet documentation"}),"."]}),"\n",(0,i.jsx)(n.p,{children:"There are a couple core benefits to using Eraser for image cleanup:"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:"Eraser can be configured to use image vulnerability data when making determinations on image removal"}),"\n",(0,i.jsx)(n.li,{children:"By interfacing directly with the container runtime, Eraser can clean up images that are not managed by Kubelet and Kubernetes"}),"\n"]})]})}function d(e={}){const{wrapper:n}={...(0,s.R)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(u,{...e})}):u(e)}},8453:(e,n,t)=>{t.d(n,{R:()=>a,x:()=>o});var r=t(6540);const i={},s=r.createContext(i);function a(e){const n=r.useContext(s);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function o(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:a(e.components),r.createElement(s.Provider,{value:n},e.children)}}}]);