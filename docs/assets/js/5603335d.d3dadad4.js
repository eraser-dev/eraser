"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[5476],{713:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>i,contentTitle:()=>c,default:()=>u,frontMatter:()=>a,metadata:()=>r,toc:()=>l});const r=JSON.parse('{"id":"custom-scanner","title":"Custom Scanner","description":"Creating a Custom Scanner","source":"@site/docs/custom-scanner.md","sourceDirName":".","slug":"/custom-scanner","permalink":"/eraser/docs/next/custom-scanner","draft":false,"unlisted":false,"tags":[],"version":"current","frontMatter":{"title":"Custom Scanner"},"sidebar":"sidebar","previous":{"title":"Releasing","permalink":"/eraser/docs/next/releasing"},"next":{"title":"Trivy","permalink":"/eraser/docs/next/trivy"}}');var s=t(4848),o=t(8453);const a={title:"Custom Scanner"},c=void 0,i={},l=[{value:"Creating a Custom Scanner",id:"creating-a-custom-scanner",level:2}];function d(e){const n={a:"a",code:"code",h2:"h2",p:"p",...(0,o.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.h2,{id:"creating-a-custom-scanner",children:"Creating a Custom Scanner"}),"\n",(0,s.jsxs)(n.p,{children:["To create a custom scanner for non-compliant images, use the following ",(0,s.jsx)(n.a,{href:"https://github.com/eraser-dev/eraser-scanner-template/",children:"template"}),"."]}),"\n",(0,s.jsxs)(n.p,{children:["In order to customize your scanner, start by creating a ",(0,s.jsx)(n.code,{children:"NewImageProvider()"}),". The ImageProvider interface can be found can be found ",(0,s.jsx)(n.a,{target:"_blank","data-noBrokenLinkCheck":!0,href:t(4952).A+"",children:"here"}),"."]}),"\n",(0,s.jsxs)(n.p,{children:["The ImageProvider will allow you to retrieve the list of all non-running and non-excluded images from the collector container through the ",(0,s.jsx)(n.code,{children:"ReceiveImages()"})," function. Process these images with your customized scanner and threshold, and use ",(0,s.jsx)(n.code,{children:"SendImages()"})," to pass the images found non-compliant to the eraser container for removal. Finally, complete the scanning process by calling ",(0,s.jsx)(n.code,{children:"Finish()"}),"."]}),"\n",(0,s.jsx)(n.p,{children:"When complete, provide your custom scanner image to Eraser in deployment."})]})}function u(e={}){const{wrapper:n}={...(0,o.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(d,{...e})}):d(e)}},4952:(e,n,t)=>{t.d(n,{A:()=>r});const r=t.p+"assets/files/scanner_template-1354bd0e962dd16dc5001599d249b071.go"},8453:(e,n,t)=>{t.d(n,{R:()=>a,x:()=>c});var r=t(6540);const s={},o=r.createContext(s);function a(e){const n=r.useContext(o);return r.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function c(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:a(e.components),r.createElement(o.Provider,{value:n},e.children)}}}]);