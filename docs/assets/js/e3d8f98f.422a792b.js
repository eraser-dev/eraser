"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[6236],{3905:(e,t,r)=>{r.d(t,{Zo:()=>p,kt:()=>d});var n=r(7294);function o(e,t,r){return t in e?Object.defineProperty(e,t,{value:r,enumerable:!0,configurable:!0,writable:!0}):e[t]=r,e}function a(e,t){var r=Object.keys(e);if(Object.getOwnPropertySymbols){var n=Object.getOwnPropertySymbols(e);t&&(n=n.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),r.push.apply(r,n)}return r}function i(e){for(var t=1;t<arguments.length;t++){var r=null!=arguments[t]?arguments[t]:{};t%2?a(Object(r),!0).forEach((function(t){o(e,t,r[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(r)):a(Object(r)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(r,t))}))}return e}function l(e,t){if(null==e)return{};var r,n,o=function(e,t){if(null==e)return{};var r,n,o={},a=Object.keys(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||(o[r]=e[r]);return o}(e,t);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);for(n=0;n<a.length;n++)r=a[n],t.indexOf(r)>=0||Object.prototype.propertyIsEnumerable.call(e,r)&&(o[r]=e[r])}return o}var c=n.createContext({}),s=function(e){var t=n.useContext(c),r=t;return e&&(r="function"==typeof e?e(t):i(i({},t),e)),r},p=function(e){var t=s(e.components);return n.createElement(c.Provider,{value:t},e.children)},m={inlineCode:"code",wrapper:function(e){var t=e.children;return n.createElement(n.Fragment,{},t)}},u=n.forwardRef((function(e,t){var r=e.components,o=e.mdxType,a=e.originalType,c=e.parentName,p=l(e,["components","mdxType","originalType","parentName"]),u=s(r),d=o,f=u["".concat(c,".").concat(d)]||u[d]||m[d]||a;return r?n.createElement(f,i(i({ref:t},p),{},{components:r})):n.createElement(f,i({ref:t},p))}));function d(e,t){var r=arguments,o=t&&t.mdxType;if("string"==typeof e||o){var a=r.length,i=new Array(a);i[0]=u;var l={};for(var c in t)hasOwnProperty.call(t,c)&&(l[c]=t[c]);l.originalType=e,l.mdxType="string"==typeof e?e:o,i[1]=l;for(var s=2;s<a;s++)i[s]=r[s];return n.createElement.apply(null,i)}return n.createElement.apply(null,r)}u.displayName="MDXCreateElement"},6040:(e,t,r)=>{r.r(t),r.d(t,{assets:()=>c,contentTitle:()=>i,default:()=>m,frontMatter:()=>a,metadata:()=>l,toc:()=>s});var n=r(7462),o=(r(7294),r(3905));const a={title:"Metrics"},i=void 0,l={unversionedId:"metrics",id:"version-v1.2.x/metrics",title:"Metrics",description:"To view Eraser metrics, you will need to deploy an Open Telemetry collector in the 'eraser-system' namespace, and an exporter. An example collector with a Prometheus exporter is otelcollector.yaml, and the endpoint can be specified using the configmap. In this example, we are logging the collected data to the otel-collector pod, and exporting metrics through Prometheus at 'http8889/metrics', but a separate exporter can also be configured.",source:"@site/versioned_docs/version-v1.2.x/metrics.md",sourceDirName:".",slug:"/metrics",permalink:"/eraser/docs/metrics",draft:!1,tags:[],version:"v1.2.x",frontMatter:{title:"Metrics"},sidebar:"sidebar",previous:{title:"Customization",permalink:"/eraser/docs/customization"},next:{title:"Setup",permalink:"/eraser/docs/setup"}},c={},s=[{value:"Eraser",id:"eraser",level:4},{value:"Scanner",id:"scanner",level:4},{value:"ImageJob",id:"imagejob",level:4}],p={toc:s};function m(e){let{components:t,...r}=e;return(0,o.kt)("wrapper",(0,n.Z)({},p,r,{components:t,mdxType:"MDXLayout"}),(0,o.kt)("p",null,"To view Eraser metrics, you will need to deploy an Open Telemetry collector in the 'eraser-system' namespace, and an exporter. An example collector with a Prometheus exporter is ",(0,o.kt)("a",{parentName:"p",href:"https://github.com/eraser-dev/eraser/blob/main/test/e2e/test-data/otelcollector.yaml"},"otelcollector.yaml"),", and the endpoint can be specified using the ",(0,o.kt)("a",{parentName:"p",href:"https://eraser-dev.github.io/eraser/docs/customization#universal-options"},"configmap"),". In this example, we are logging the collected data to the otel-collector pod, and exporting metrics through Prometheus at 'http://localhost:8889/metrics', but a separate exporter can also be configured."),(0,o.kt)("p",null,"Below is the list of metrics provided by Eraser per run:"),(0,o.kt)("h4",{id:"eraser"},"Eraser"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},"- count\n    - name: images_removed_run_total\n        - description: Total images removed by eraser\n")),(0,o.kt)("h4",{id:"scanner"},"Scanner"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},"- count\n    - name: vulnerable_images_run_total\n        - description: Total vulnerable images detected\n")),(0,o.kt)("h4",{id:"imagejob"},"ImageJob"),(0,o.kt)("pre",null,(0,o.kt)("code",{parentName:"pre",className:"language-yaml"},"- count\n    - name: imagejob_run_total\n        - description: Total ImageJobs scheduled\n    - name: pods_completed_run_total\n        - description: Total pods completed\n    -  name: pods_failed_run_total\n        - description: Total pods failed\n- summary\n    - name: imagejob_duration_run_seconds\n        - description: Total time for ImageJobs scheduled to complete\n")))}m.isMDXComponent=!0}}]);