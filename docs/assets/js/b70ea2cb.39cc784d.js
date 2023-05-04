"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[2534],{3905:(e,t,r)=>{r.d(t,{Zo:()=>p,kt:()=>g});var n=r(7294);function a(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function i(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function l(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?i(Object(r),!0).forEach((function(t){a(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):i(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function o(e,t){if(null==e)return{};var r,n,a=function(e,t){if(null==e)return{};var r,n,a={},i=Object.keys(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||(a[r]=e[r]);return a}(e,t);if(Object.getOwnPropertySymbols){var i=Object.getOwnPropertySymbols(e);for(n=0;n<i.length;n++)r=i[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(a[r]=e[r])}return a}var s=n.createContext({}),c=function(e){var t=n.useContext(s),r=t;return e&&(r="function"==typeof e?e(t):l(l({},t),e)),r},p=function(e){var t=c(e.components);return n.createElement(s.Provider,{value:t},e.children)},u={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},m=n.forwardRef((function(e,t){var r=e.components,a=e.mdxType,i=e.originalType,s=e.parentName,p=o(e,["components","mdxType","originalType","parentName"]),m=c(r),g=a,d=m["".concat(s,".").concat(g)]||m[g]||u[g]||i;return r?n.createElement(d,l(l({ref:t},p),{},{components:r})):n.createElement(d,l({ref:t},p))}));function g(e,t){var r=arguments,a=t&&t.mdxType;if("string"==typeof e||a){var i=r.length,l=new Array(i);l[0]=m;var o={};for(var s in t)hasOwnProperty.call(t,s)&&(o[s]=t[s]);o.originalType=e,o.mdxType="string"==typeof e?e:a,l[1]=o;for(var c=2;c<i;c++)l[c]=r[c];return n.createElement.apply(null,l)}return n.createElement.apply(null,r)}m.displayName="MDXCreateElement"},4446:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>s,contentTitle:()=>l,default:()=>u,frontMatter:()=>i,metadata:()=>o,toc:()=>c});var n=r(7462),a=(r(7294),r(3905));const i={title:"Releasing"},l=void 0,o={unversionedId:"releasing",id:"version-v1.1.x/releasing",title:"Releasing",description:"Create Release Pull Request",source:"@site/versioned_docs/version-v1.1.x/releasing.md",sourceDirName:".",slug:"/releasing",permalink:"/eraser/docs/releasing",draft:!1,tags:[],version:"v1.1.x",frontMatter:{title:"Releasing"}},s={},c=[{value:"Create Release Pull Request",id:"create-release-pull-request",level:2},{value:"Publishing",id:"publishing",level:2}],p={toc:c};function u(e){let{components:t,...r}=e;return(0,a.kt)("wrapper",(0,n.Z)({},p,r,{components:t,mdxType:"MDXLayout"}),(0,a.kt)("h2",{id:"create-release-pull-request"},"Create Release Pull Request"),(0,a.kt)("ol",null,(0,a.kt)("li",{parentName:"ol"},"Go to ",(0,a.kt)("inlineCode",{parentName:"li"},"create_release_pull_request")," workflow under actions."),(0,a.kt)("li",{parentName:"ol"},"Select run workflow, and use the workflow from your branch. "),(0,a.kt)("li",{parentName:"ol"},"Input release version with the semantic version identifying the release."),(0,a.kt)("li",{parentName:"ol"},"Click run workflow and review the PR created by github-actions.")),(0,a.kt)("h1",{id:"releasing"},"Releasing"),(0,a.kt)("ol",{start:5},(0,a.kt)("li",{parentName:"ol"},(0,a.kt)("p",{parentName:"li"},"Once the PR is merged to ",(0,a.kt)("inlineCode",{parentName:"p"},"main"),", tag that commit with release version and push tags to remote repository."),(0,a.kt)("pre",{parentName:"li"},(0,a.kt)("code",{parentName:"pre"},"git checkout <BRANCH NAME>\ngit pull origin <BRANCH NAME>\ngit tag -a <NEW VERSION> -m '<NEW VERSION>'\ngit push origin <NEW VERSION>\n"))),(0,a.kt)("li",{parentName:"ol"},(0,a.kt)("p",{parentName:"li"},"Pushing the release tag will trigger GitHub Actions to trigger ",(0,a.kt)("inlineCode",{parentName:"p"},"release")," job.\nThis will build the ",(0,a.kt)("inlineCode",{parentName:"p"},"ghcr.io/azure/remover"),", ",(0,a.kt)("inlineCode",{parentName:"p"},"ghcr.io/azure/eraser-manager"),", ",(0,a.kt)("inlineCode",{parentName:"p"},"ghcr.io/azure/collector"),", and ",(0,a.kt)("inlineCode",{parentName:"p"},"ghcr.io/azure/eraser-trivy-scanner")," images automatically, then publish the new release tag."))),(0,a.kt)("h2",{id:"publishing"},"Publishing"),(0,a.kt)("ol",null,(0,a.kt)("li",{parentName:"ol"},"GitHub Action will create a new release, review and edit it at ",(0,a.kt)("a",{parentName:"li",href:"https://github.com/Azure/eraser/releases"},"https://github.com/Azure/eraser/releases"))))}u.isMDXComponent=!0}}]);