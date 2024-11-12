"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4008],{510:(e,s,r)=>{r.r(s),r.d(s,{assets:()=>d,contentTitle:()=>t,default:()=>h,frontMatter:()=>o,metadata:()=>i,toc:()=>c});const i=JSON.parse('{"id":"exclusion","title":"Exclusion","description":"Excluding registries, repositories, and images","source":"@site/versioned_docs/version-v0.5.x/exclusion.md","sourceDirName":".","slug":"/exclusion","permalink":"/eraser/docs/v0.5.x/exclusion","draft":false,"unlisted":false,"tags":[],"version":"v0.5.x","frontMatter":{"title":"Exclusion"},"sidebar":"sidebar","previous":{"title":"Manual Removal","permalink":"/eraser/docs/v0.5.x/manual-removal"},"next":{"title":"Customization","permalink":"/eraser/docs/v0.5.x/customization"}}');var n=r(4848),l=r(8453);const o={title:"Exclusion"},t=void 0,d={},c=[{value:"Excluding registries, repositories, and images",id:"excluding-registries-repositories-and-images",level:2},{value:"Exempting Nodes from the Eraser Pipeline",id:"exempting-nodes-from-the-eraser-pipeline",level:2}];function a(e){const s={a:"a",code:"code",em:"em",h2:"h2",li:"li",p:"p",pre:"pre",ul:"ul",...(0,l.R)(),...e.components};return(0,n.jsxs)(n.Fragment,{children:[(0,n.jsx)(s.h2,{id:"excluding-registries-repositories-and-images",children:"Excluding registries, repositories, and images"}),"\n",(0,n.jsxs)(s.p,{children:["Eraser can exclude registries (example, ",(0,n.jsx)(s.code,{children:"docker.io/library/*"}),") and also specific images with a tag (example, ",(0,n.jsx)(s.code,{children:"docker.io/library/ubuntu:18.04"}),") or digest (example, ",(0,n.jsx)(s.code,{children:"sha256:80f31da1ac7b312ba29d65080fd..."}),") from its removal process."]}),"\n",(0,n.jsxs)(s.p,{children:["To exclude any images or registries from the removal, create configmap(s) with the label ",(0,n.jsx)(s.code,{children:"eraser.sh/exclude.list=true"})," in the eraser-system namespace with a JSON file holding the excluded images."]}),"\n",(0,n.jsx)(s.pre,{children:(0,n.jsx)(s.code,{className:"language-bash",children:'$ cat > sample.json <<EOF\n{"excluded": ["docker.io/library/*", "ghcr.io/eraser-dev/test:latest"]}\nEOF\n\n$ kubectl create configmap excluded --from-file=sample.json --namespace=eraser-system\n$ kubectl label configmap excluded eraser.sh/exclude.list=true -n eraser-system\n'})}),"\n",(0,n.jsx)(s.h2,{id:"exempting-nodes-from-the-eraser-pipeline",children:"Exempting Nodes from the Eraser Pipeline"}),"\n",(0,n.jsxs)(s.p,{children:["Exempting nodes with ",(0,n.jsx)(s.code,{children:"--filter-nodes"})," is added in v0.3.0. When deploying Eraser, you can specify whether there is a list of nodes you would like to ",(0,n.jsx)(s.code,{children:"include"})," or ",(0,n.jsx)(s.code,{children:"exclude"})," from the cleanup process using the ",(0,n.jsx)(s.code,{children:"--filter-nodes"})," argument."]}),"\n",(0,n.jsx)(s.p,{children:(0,n.jsxs)(s.em,{children:["See ",(0,n.jsx)(s.a,{href:"https://github.com/eraser-dev/eraser/blob/main/charts/eraser/README.md",children:"Eraser Helm Chart"})," for more information on deployment."]})}),"\n",(0,n.jsxs)(s.p,{children:["Nodes with the selector ",(0,n.jsx)(s.code,{children:"eraser.sh/cleanup.filter"})," will be filtered accordingly."]}),"\n",(0,n.jsxs)(s.ul,{children:["\n",(0,n.jsxs)(s.li,{children:["If ",(0,n.jsx)(s.code,{children:"include"})," is provided, eraser and collector pods will only be scheduled on nodes with the selector ",(0,n.jsx)(s.code,{children:"eraser.sh/cleanup.filter"}),"."]}),"\n",(0,n.jsxs)(s.li,{children:["If ",(0,n.jsx)(s.code,{children:"exclude"})," is provided, eraser and collector pods will be scheduled on all nodes besides those with the selector ",(0,n.jsx)(s.code,{children:"eraser.sh/cleanup.filter"}),"."]}),"\n"]}),"\n",(0,n.jsxs)(s.p,{children:["Unless specified, the default value of ",(0,n.jsx)(s.code,{children:"--filter-nodes"})," is ",(0,n.jsx)(s.code,{children:"exclude"}),". Because Windows nodes are not supported, they will always be excluded regardless of the ",(0,n.jsx)(s.code,{children:"eraser.sh/cleanup.filter"})," label or the value of ",(0,n.jsx)(s.code,{children:"--filter-nodes"}),"."]}),"\n",(0,n.jsxs)(s.p,{children:["Additional node selectors can be provided through the ",(0,n.jsx)(s.code,{children:"--filter-nodes-selector"})," flag."]})]})}function h(e={}){const{wrapper:s}={...(0,l.R)(),...e.components};return s?(0,n.jsx)(s,{...e,children:(0,n.jsx)(a,{...e})}):a(e)}},8453:(e,s,r)=>{r.d(s,{R:()=>o,x:()=>t});var i=r(6540);const n={},l=i.createContext(n);function o(e){const s=i.useContext(l);return i.useMemo((function(){return"function"==typeof e?e(s):{...s,...e}}),[s,e])}function t(e){let s;return s=e.disableParentContext?"function"==typeof e.components?e.components(n):e.components||n:o(e.components),i.createElement(l.Provider,{value:s},e.children)}}}]);