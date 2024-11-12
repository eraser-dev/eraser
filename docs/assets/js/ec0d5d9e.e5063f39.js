"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[8018],{8078:(e,a,n)=>{n.r(a),n.d(a,{assets:()=>o,contentTitle:()=>l,default:()=>d,frontMatter:()=>i,metadata:()=>t,toc:()=>c});const t=JSON.parse('{"id":"architecture","title":"Architecture","description":"At a high level, Eraser has two main modes of operation: manual and automated.","source":"@site/versioned_docs/version-v0.5.x/architecture.md","sourceDirName":".","slug":"/architecture","permalink":"/eraser/docs/v0.5.x/architecture","draft":false,"unlisted":false,"tags":[],"version":"v0.5.x","frontMatter":{"title":"Architecture"},"sidebar":"sidebar","previous":{"title":"Quick Start","permalink":"/eraser/docs/v0.5.x/quick-start"},"next":{"title":"Manual Removal","permalink":"/eraser/docs/v0.5.x/manual-removal"}}');var r=n(4848),s=n(8453);const i={title:"Architecture"},l=void 0,o={},c=[{value:"Manual image cleanup",id:"manual-image-cleanup",level:2},{value:"Automated analysis, scanning, and cleanup",id:"automated-analysis-scanning-and-cleanup",level:2}];function u(e){const a={h2:"h2",p:"p",...(0,s.R)(),...e.components};return(0,r.jsxs)(r.Fragment,{children:[(0,r.jsx)(a.p,{children:"At a high level, Eraser has two main modes of operation: manual and automated."}),"\n",(0,r.jsx)(a.p,{children:"Manual image removal involves supplying a list of images to remove; Eraser then\ndeploys pods to clean up the images you supplied."}),"\n",(0,r.jsx)(a.p,{children:"Automated image removal runs on a timer. By default, the automated process\nremoves images based on the results of a vulnerability scan. The default\nvulnerability scanner is Trivy, but others can be provided in its place. Or,\nthe scanner can be disabled altogether, in which case Eraser acts as a garbage\ncollector -- it will remove all non-running images in your cluster."}),"\n",(0,r.jsx)(a.h2,{id:"manual-image-cleanup",children:"Manual image cleanup"}),"\n",(0,r.jsx)(a.p,{children:"Note: metrics are not yet implemented in Eraser v0.5.x, but will be available in the upcoming v1.0.0 release."}),"\n",(0,r.jsx)("img",{title:"manual cleanup",src:"/eraser/docs/img/eraser_manual.png"}),"\n",(0,r.jsx)(a.h2,{id:"automated-analysis-scanning-and-cleanup",children:"Automated analysis, scanning, and cleanup"}),"\n",(0,r.jsx)("img",{title:"automated cleanup",src:"/eraser/docs/img/eraser_timer.png"})]})}function d(e={}){const{wrapper:a}={...(0,s.R)(),...e.components};return a?(0,r.jsx)(a,{...e,children:(0,r.jsx)(u,{...e})}):u(e)}},8453:(e,a,n)=>{n.d(a,{R:()=>i,x:()=>l});var t=n(6540);const r={},s=t.createContext(r);function i(e){const a=t.useContext(s);return t.useMemo((function(){return"function"==typeof e?e(a):{...a,...e}}),[a,e])}function l(e){let a;return a=e.disableParentContext?"function"==typeof e.components?e.components(r):e.components||r:i(e.components),t.createElement(s.Provider,{value:a},e.children)}}}]);