"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[9829],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>d});var r=n(7294);function a(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function o(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);t&&(r=r.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,r)}return n}function c(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?o(Object(n),!0).forEach((function(t){a(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):o(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function i(e,t){if(null==e)return{};var n,r,a=function(e,t){if(null==e)return{};var n,r,a={},o=Object.keys(e);for(r=0;r<o.length;r++)n=o[r],t.indexOf(n)>=0||(a[n]=e[n]);return a}(e,t);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);for(r=0;r<o.length;r++)n=o[r],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(a[n]=e[n])}return a}var s=r.createContext({}),l=function(e){var t=r.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):c(c({},t),e)),n},p=function(e){var t=l(e.components);return r.createElement(s.Provider,{value:t},e.children)},u={inlineCode:"code",wrapper:function(e){var t=e.children;return r.createElement(r.Fragment,{},t)}},m=r.forwardRef((function(e,t){var n=e.components,a=e.mdxType,o=e.originalType,s=e.parentName,p=i(e,["components","mdxType","originalType","parentName"]),m=l(n),d=a,f=m["".concat(s,".").concat(d)]||m[d]||u[d]||o;return n?r.createElement(f,c(c({ref:t},p),{},{components:n})):r.createElement(f,c({ref:t},p))}));function d(e,t){var n=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var o=n.length,c=new Array(o);c[0]=m;var i={};for(var s in t)hasOwnProperty.call(t,s)&&(i[s]=t[s]);i.originalType=e,i.mdxType="string"==typeof e?e:a,c[1]=i;for(var l=2;l<o;l++)c[l]=n[l];return r.createElement.apply(null,c)}return r.createElement.apply(null,n)}m.displayName="MDXCreateElement"},4129:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>s,contentTitle:()=>c,default:()=>u,frontMatter:()=>o,metadata:()=>i,toc:()=>l});var r=n(7462),a=(n(7294),n(3905));const o={title:"Custom Scanner"},c=void 0,i={unversionedId:"custom-scanner",id:"version-v1.3.x/custom-scanner",title:"Custom Scanner",description:"Creating a Custom Scanner",source:"@site/versioned_docs/version-v1.3.x/custom-scanner.md",sourceDirName:".",slug:"/custom-scanner",permalink:"/eraser/docs/custom-scanner",draft:!1,tags:[],version:"v1.3.x",frontMatter:{title:"Custom Scanner"},sidebar:"sidebar",previous:{title:"Releasing",permalink:"/eraser/docs/releasing"},next:{title:"Trivy",permalink:"/eraser/docs/trivy"}},s={},l=[{value:"Creating a Custom Scanner",id:"creating-a-custom-scanner",level:2}],p={toc:l};function u(e){let{components:t,...n}=e;return(0,a.kt)("wrapper",(0,r.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,a.kt)("h2",{id:"creating-a-custom-scanner"},"Creating a Custom Scanner"),(0,a.kt)("p",null,"To create a custom scanner for non-compliant images, use the following ",(0,a.kt)("a",{parentName:"p",href:"https://github.com/eraser-dev/eraser-scanner-template/"},"template"),"."),(0,a.kt)("p",null,"In order to customize your scanner, start by creating a ",(0,a.kt)("inlineCode",{parentName:"p"},"NewImageProvider()"),". The ImageProvider interface can be found can be found ",(0,a.kt)("a",{parentName:"p",href:"../../pkg/scanners/template/scanner_template.go"},"here"),". "),(0,a.kt)("p",null,"The ImageProvider will allow you to retrieve the list of all non-running and non-excluded images from the collector container through the ",(0,a.kt)("inlineCode",{parentName:"p"},"ReceiveImages()")," function. Process these images with your customized scanner and threshold, and use ",(0,a.kt)("inlineCode",{parentName:"p"},"SendImages()")," to pass the images found non-compliant to the eraser container for removal. Finally, complete the scanning process by calling ",(0,a.kt)("inlineCode",{parentName:"p"},"Finish()"),"."),(0,a.kt)("p",null,"When complete, provide your custom scanner image to Eraser in deployment."))}u.isMDXComponent=!0}}]);