"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[5243],{4028:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>c,contentTitle:()=>r,default:()=>u,frontMatter:()=>i,metadata:()=>a,toc:()=>d});var o=n(5893),s=n(1151);const i={title:"Introduction",slug:"/"},r="Introduction",a={id:"introduction",title:"Introduction",description:"When deploying to Kubernetes, it's common for pipelines to build and push images to a cluster, but it's much less common for these images to be cleaned up. This can lead to accumulating bloat on the disk, and a host of non-compliant images lingering on the nodes.",source:"@site/versioned_docs/version-v1.1.x/introduction.md",sourceDirName:".",slug:"/",permalink:"/eraser/docs/v1.1.x/",draft:!1,unlisted:!1,tags:[],version:"v1.1.x",frontMatter:{title:"Introduction",slug:"/"},sidebar:"sidebar",next:{title:"Installation",permalink:"/eraser/docs/v1.1.x/installation"}},c={},d=[];function l(e){const t={h1:"h1",header:"header",p:"p",strong:"strong",...(0,s.a)(),...e.components};return(0,o.jsxs)(o.Fragment,{children:[(0,o.jsx)(t.header,{children:(0,o.jsx)(t.h1,{id:"introduction",children:"Introduction"})}),"\n",(0,o.jsx)(t.p,{children:"When deploying to Kubernetes, it's common for pipelines to build and push images to a cluster, but it's much less common for these images to be cleaned up. This can lead to accumulating bloat on the disk, and a host of non-compliant images lingering on the nodes."}),"\n",(0,o.jsxs)(t.p,{children:["The current garbage collection process deletes images based on a percentage of load, but this process does not consider the vulnerability state of the images. ",(0,o.jsx)(t.strong,{children:"Eraser"})," aims to provide a simple way to determine the state of an image, and delete it if it meets the specified criteria."]})]})}function u(e={}){const{wrapper:t}={...(0,s.a)(),...e.components};return t?(0,o.jsx)(t,{...e,children:(0,o.jsx)(l,{...e})}):l(e)}},1151:(e,t,n)=>{n.d(t,{Z:()=>a,a:()=>r});var o=n(7294);const s={},i=o.createContext(s);function r(e){const t=o.useContext(i);return o.useMemo((function(){return"function"==typeof e?e(t):{...t,...e}}),[t,e])}function a(e){let t;return t=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:r(e.components),o.createElement(i.Provider,{value:t},e.children)}}}]);