"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[5873],{3905:(e,n,t)=>{t.d(n,{Zo:()=>u,kt:()=>f});var r=t(7294);function o(e,n,t){return n in e?Object.defineProperty(e,n,{value:t,enumerable:!0,configurable:!0,writable:!0}):e[n]=t,e}function a(e,n){var t=Object.keys(e);if(Object.getOwnPropertySymbols){var r=Object.getOwnPropertySymbols(e);n&&(r=r.filter((function(n){return Object.getOwnPropertyDescriptor(e,n).enumerable}))),t.push.apply(t,r)}return t}function c(e){for(var n=1;n<arguments.length;n++){var t=null!=arguments[n]?arguments[n]:{};n%2?a(Object(t),!0).forEach((function(n){o(e,n,t[n])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(t)):a(Object(t)).forEach((function(n){Object.defineProperty(e,n,Object.getOwnPropertyDescriptor(t,n))}))}return e}function i(e,n){if(null==e)return{};var t,r,o=function(e,n){if(null==e)return{};var t,r,o={},a=Object.keys(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||(o[t]=e[t]);return o}(e,n);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(r=0;r<a.length;r++)t=a[r],n.indexOf(t)>=0||Object.prototype.propertyIsEnumerable.call(e,t)&&(o[t]=e[t])}return o}var s=r.createContext({}),l=function(e){var n=r.useContext(s),t=n;return e&&(t="function"==typeof e?e(n):c(c({},n),e)),t},u=function(e){var n=l(e.components);return r.createElement(s.Provider,{value:n},e.children)},p={inlineCode:"code",wrapper:function(e){var n=e.children;return r.createElement(r.Fragment,{},n)}},m=r.forwardRef((function(e,n){var t=e.components,o=e.mdxType,a=e.originalType,s=e.parentName,u=i(e,["components","mdxType","originalType","parentName"]),m=l(t),f=o,d=m["".concat(s,".").concat(f)]||m[f]||p[f]||a;return t?r.createElement(d,c(c({ref:n},u),{},{components:t})):r.createElement(d,c({ref:n},u))}));function f(e,n){var t=arguments,o=n&&n.mdxType;if("string"==typeof e||o){var a=t.length,c=new Array(a);c[0]=m;var i={};for(var s in n)hasOwnProperty.call(n,s)&&(i[s]=n[s]);i.originalType=e,i.mdxType="string"==typeof e?e:o,c[1]=i;for(var l=2;l<a;l++)c[l]=t[l];return r.createElement.apply(null,c)}return r.createElement.apply(null,t)}m.displayName="MDXCreateElement"},1261:(e,n,t)=>{t.r(n),t.d(n,{assets:()=>s,contentTitle:()=>c,default:()=>p,frontMatter:()=>a,metadata:()=>i,toc:()=>l});var r=t(7462),o=(t(7294),t(3905));const a={title:"Custom Scanner"},c=void 0,i={unversionedId:"custom-scanner",id:"version-v1.1.x/custom-scanner",title:"Custom Scanner",description:"Creating a Custom Scanner",source:"@site/versioned_docs/version-v1.1.x/custom-scanner.md",sourceDirName:".",slug:"/custom-scanner",permalink:"/eraser/docs/custom-scanner",draft:!1,tags:[],version:"v1.1.x",frontMatter:{title:"Custom Scanner"}},s={},l=[{value:"Creating a Custom Scanner",id:"creating-a-custom-scanner",level:2}],u={toc:l};function p(e){let{components:n,...t}=e;return(0,o.kt)("wrapper",(0,r.Z)({},u,t,{components:n,mdxType:"MDXLayout"}),(0,o.kt)("h2",{id:"creating-a-custom-scanner"},"Creating a Custom Scanner"),(0,o.kt)("p",null,"To create a custom scanner for non-compliant images, use the following ",(0,o.kt)("a",{parentName:"p",href:"https://github.com/Azure/eraser-scanner-template/"},"template"),"."),(0,o.kt)("p",null,"In order to customize your scanner, start by creating a ",(0,o.kt)("inlineCode",{parentName:"p"},"NewImageProvider()"),". The ImageProvider interface can be found can be found ",(0,o.kt)("a",{parentName:"p",href:"../../pkg/scanners/template/scanner_template.go"},"here"),". "),(0,o.kt)("p",null,"The ImageProvider will allow you to retrieve the list of all non-running and non-excluded images from the collector container through the ",(0,o.kt)("inlineCode",{parentName:"p"},"ReceiveImages()")," function. Process these images with your customized scanner and threshold, and use ",(0,o.kt)("inlineCode",{parentName:"p"},"SendImages()")," to pass the images found non-compliant to the eraser container for removal. Finally, complete the scanning process by calling ",(0,o.kt)("inlineCode",{parentName:"p"},"Finish()"),"."),(0,o.kt)("p",null,"When complete, provide your custom scanner image to Eraser in deployment."))}p.isMDXComponent=!0}}]);