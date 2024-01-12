"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4611],{3905:(e,t,r)=>{r.d(t,{Zo:()=>u,kt:()=>b});var n=r(7294);function a(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function i(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function o(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?i(Object(r),!0).forEach((function(t){a(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):i(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function l(e,t){if(null==e)return{};var r,n,a=function(e,t){if(null==e)return{};var r,n,a={},i=Object.keys(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||(a[r]=e[r]);return a}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(a[r]=e[r])}return a}var s=n.createContext({}),c=function(e){var t=n.useContext(s),r=t;return e&&(r="function"==typeof e?e(t):o(o({},t),e)),r},u=function(e){var t=c(e.components);return n.createElement(s.Provider,{value:t},e.children)},p={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},f=n.forwardRef((function(e,t){var r=e.components,a=e.mdxType,i=e.originalType,s=e.parentName,u=l(e,["components","mdxType","originalType","parentName"]),f=c(r),b=a,g=f["".concat(s,".").concat(b)]||f[b]||p[b]||i;return r?n.createElement(g,o(o({ref:t},u),{},{components:r})):n.createElement(g,o({ref:t},u))}));function b(e,t){var r=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var i=r.length,o=new Array(i);o[0]=f;var l={};for(var s in t)hasOwnProperty.call(t,s)&&(l[s]=t[s]);l.originalType=e,l.mdxType="string"==typeof e?e:a,o[1]=l;for(var c=2;c<i;c++)o[c]=r[c];return n.createElement.apply(null,o)}return n.createElement.apply(null,r)}f.displayName="MDXCreateElement"},1656:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>s,contentTitle:()=>o,default:()=>p,frontMatter:()=>i,metadata:()=>l,toc:()=>c});var n=r(7462),a=(r(7294),r(3905));const i={title:"FAQ"},o=void 0,l={unversionedId:"faq",id:"version-v1.4.0-beta.0/faq",title:"FAQ",description:"Why am I still seeing vulnerable images?",source:"@site/versioned_docs/version-v1.4.0-beta.0/faq.md",sourceDirName:".",slug:"/faq",permalink:"/eraser/docs/faq",draft:!1,tags:[],version:"v1.4.0-beta.0",frontMatter:{title:"FAQ"},sidebar:"sidebar",previous:{title:"Trivy",permalink:"/eraser/docs/trivy"},next:{title:"Contributing",permalink:"/eraser/docs/contributing"}},s={},c=[{value:"Why am I still seeing vulnerable images?",id:"why-am-i-still-seeing-vulnerable-images",level:2},{value:"How is Eraser different from Kubernetes garbage collection?",id:"how-is-eraser-different-from-kubernetes-garbage-collection",level:2}],u={toc:c};function p(e){let{components:t,...r}=e;return(0,a.kt)("wrapper",(0,n.Z)({},u,r,{components:t,mdxType:"MDXLayout"}),(0,a.kt)("h2",{id:"why-am-i-still-seeing-vulnerable-images"},"Why am I still seeing vulnerable images?"),(0,a.kt)("p",null,"Eraser currently targets ",(0,a.kt)("strong",{parentName:"p"},"non-running")," images, so any vulnerable images that are currently running will not be removed. In addition, the default vulnerability scanning with Trivy removes images with ",(0,a.kt)("inlineCode",{parentName:"p"},"CRITICAL")," vulnerabilities. Any images with lower vulnerabilities will not be removed. This can be configured using the ",(0,a.kt)("a",{parentName:"p",href:"https://eraser-dev.github.io/eraser/docs/customization#scanner-options"},"configmap"),"."),(0,a.kt)("h2",{id:"how-is-eraser-different-from-kubernetes-garbage-collection"},"How is Eraser different from Kubernetes garbage collection?"),(0,a.kt)("p",null,"The native garbage collection in Kubernetes works a bit differently than Eraser. By default, garbage collection begins when disk usage reaches 85%, and stops when it gets down to 80%. More details about Kubernetes garbage collection can be found in the ",(0,a.kt)("a",{parentName:"p",href:"https://kubernetes.io/docs/concepts/architecture/garbage-collection/"},"Kubernetes documentation"),", and configuration options can be found in the ",(0,a.kt)("a",{parentName:"p",href:"https://kubernetes.io/docs/reference/config-api/kubelet-config.v1beta1/"},"Kubelet documentation"),". "),(0,a.kt)("p",null,"There are a couple core benefits to using Eraser for image cleanup:"),(0,a.kt)("ul",null,(0,a.kt)("li",{parentName:"ul"},"Eraser can be configured to use image vulnerability data when making determinations on image removal"),(0,a.kt)("li",{parentName:"ul"},"By interfacing directly with the container runtime, Eraser can clean up images that are not managed by Kubelet and Kubernetes")))}p.isMDXComponent=!0}}]);