"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4497],{3905:(e,t,n)=>{n.d(t,{Zo:()=>p,kt:()=>d});var a=n(7294);function r(e,t,n){return t in e?Object.defineProperty(e,t,{value:n,enumerable:!0,configurable:!0,writable:!0}):e[t]=n,e}function o(e,t){var n=Object.keys(e);if(Object.getOwnPropertySymbols){var a=Object.getOwnPropertySymbols(e);t&&(a=a.filter((function(t){return Object.getOwnPropertyDescriptor(e,t).enumerable}))),n.push.apply(n,a)}return n}function l(e){for(var t=1;t<arguments.length;t++){var n=null!=arguments[t]?arguments[t]:{};t%2?o(Object(n),!0).forEach((function(t){r(e,t,n[t])})):Object.getOwnPropertyDescriptors?Object.defineProperties(e,Object.getOwnPropertyDescriptors(n)):o(Object(n)).forEach((function(t){Object.defineProperty(e,t,Object.getOwnPropertyDescriptor(n,t))}))}return e}function i(e,t){if(null==e)return{};var n,a,r=function(e,t){if(null==e)return{};var n,a,r={},o=Object.keys(e);for(a=0;a<o.length;a++)n=o[a],t.indexOf(n)>=0||(r[n]=e[n]);return r}(e,t);if(Object.getOwnPropertySymbols){var o=Object.getOwnPropertySymbols(e);for(a=0;a<o.length;a++)n=o[a],t.indexOf(n)>=0||Object.prototype.propertyIsEnumerable.call(e,n)&&(r[n]=e[n])}return r}var s=a.createContext({}),m=function(e){var t=a.useContext(s),n=t;return e&&(n="function"==typeof e?e(t):l(l({},t),e)),n},p=function(e){var t=m(e.components);return a.createElement(s.Provider,{value:t},e.children)},u={inlineCode:"code",wrapper:function(e){var t=e.children;return a.createElement(a.Fragment,{},t)}},c=a.forwardRef((function(e,t){var n=e.components,r=e.mdxType,o=e.originalType,s=e.parentName,p=i(e,["components","mdxType","originalType","parentName"]),c=m(n),d=r,g=c["".concat(s,".").concat(d)]||c[d]||u[d]||o;return n?a.createElement(g,l(l({ref:t},p),{},{components:n})):a.createElement(g,l({ref:t},p))}));function d(e,t){var n=arguments,r=t&&t.mdxType;if("string"==typeof e||r){var o=n.length,l=new Array(o);l[0]=c;var i={};for(var s in t)hasOwnProperty.call(t,s)&&(i[s]=t[s]);i.originalType=e,i.mdxType="string"==typeof e?e:r,l[1]=i;for(var m=2;m<o;m++)l[m]=n[m];return a.createElement.apply(null,l)}return a.createElement.apply(null,n)}c.displayName="MDXCreateElement"},7325:(e,t,n)=>{n.r(t),n.d(t,{assets:()=>s,contentTitle:()=>l,default:()=>u,frontMatter:()=>o,metadata:()=>i,toc:()=>m});var a=n(7462),r=(n(7294),n(3905));const o={title:"Customization"},l=void 0,i={unversionedId:"customization",id:"customization",title:"Customization",description:"Overview",source:"@site/docs/customization.md",sourceDirName:".",slug:"/customization",permalink:"/eraser/docs/next/customization",draft:!1,tags:[],version:"current",frontMatter:{title:"Customization"},sidebar:"sidebar",previous:{title:"Exclusion",permalink:"/eraser/docs/next/exclusion"},next:{title:"Metrics",permalink:"/eraser/docs/next/metrics"}},s={},m=[{value:"Overview",id:"overview",level:2},{value:"Key Concepts",id:"key-concepts",level:2},{value:"Basic architecture",id:"basic-architecture",level:3},{value:"Scheduling",id:"scheduling",level:3},{value:"Fault Tolerance",id:"fault-tolerance",level:3},{value:"Excluding Nodes",id:"excluding-nodes",level:3},{value:"Configuring Components",id:"configuring-components",level:3},{value:"Swapping out components",id:"swapping-out-components",level:3},{value:"Universal Options",id:"universal-options",level:2},{value:"Component Options",id:"component-options",level:2},{value:"Scanner Options",id:"scanner-options",level:2},{value:"Detailed Options",id:"detailed-options",level:2}],p={toc:m};function u(e){let{components:t,...n}=e;return(0,r.kt)("wrapper",(0,a.Z)({},p,n,{components:t,mdxType:"MDXLayout"}),(0,r.kt)("h2",{id:"overview"},"Overview"),(0,r.kt)("p",null,"Eraser uses a configmap to configure its behavior. The configmap is part of the\ndeployment and it is not necessary to deploy it manually. Once deployed, the configmap\ncan be edited at any time:"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-bash"},"kubectl edit configmap --namespace eraser-system eraser-manager-config\n")),(0,r.kt)("p",null,"If an eraser job is already running, the changes will not take effect until the job completes.\nThe configuration is in yaml."),(0,r.kt)("h2",{id:"key-concepts"},"Key Concepts"),(0,r.kt)("h3",{id:"basic-architecture"},"Basic architecture"),(0,r.kt)("p",null,"The ",(0,r.kt)("em",{parentName:"p"},"manager")," runs as a pod in your cluster and manages ",(0,r.kt)("em",{parentName:"p"},"ImageJobs"),". Think of\nan ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," as a unit of work, performed on every node in your cluster. Each\nnode runs a sub-job. The goal of the ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," is to assess the images on your\ncluster's nodes, and to remove the images you don't want. There are two stages:"),(0,r.kt)("ol",null,(0,r.kt)("li",{parentName:"ol"},"Assessment"),(0,r.kt)("li",{parentName:"ol"},"Removal.")),(0,r.kt)("h3",{id:"scheduling"},"Scheduling"),(0,r.kt)("p",null,"An ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," can either be created on-demand (see ",(0,r.kt)("a",{parentName:"p",href:"https://azure.github.io/eraser/docs/manual-removal"},"Manual Removal"),"),\nor they can be spawned on a timer like a cron job. On-demand jobs skip the\nassessment stage and get right down to the business of removing the images you\nspecified. The behavior of an on-demand job is quite different from that of\ntimed jobs."),(0,r.kt)("h3",{id:"fault-tolerance"},"Fault Tolerance"),(0,r.kt)("p",null,"Because an ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," runs on every node in your cluster, and the conditions on\neach node may vary widely, some of the sub-jobs may fail. If you cannot\ntolerate any failure, set the ",(0,r.kt)("inlineCode",{parentName:"p"},"manager.imageJob.successRatio")," property to\n",(0,r.kt)("inlineCode",{parentName:"p"},"1.0"),". If 75% success sounds good to you, set it to ",(0,r.kt)("inlineCode",{parentName:"p"},"0.75"),". In that case, if\nfewer than 75% of the pods spawned by the ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," report success, the job as\na whole will be marked as a failure."),(0,r.kt)("p",null,"This is mainly to help diagnose error conditions. As such, you can set\n",(0,r.kt)("inlineCode",{parentName:"p"},"manager.imageJob.cleanup.delayOnFailure")," to a long value so that logs can be\ncaptured before the spawned pods are cleaned up."),(0,r.kt)("h3",{id:"excluding-nodes"},"Excluding Nodes"),(0,r.kt)("p",null,"For various reasons, you may want to prevent Eraser from scheduling pods on\ncertain nodes. To do so, the nodes can be given a special label. By default,\nthis label is ",(0,r.kt)("inlineCode",{parentName:"p"},"eraser.sh/cleanup.filter"),", but you can configure the behavior with\nthe options under ",(0,r.kt)("inlineCode",{parentName:"p"},"manager.nodeFilter"),". The ",(0,r.kt)("a",{parentName:"p",href:"#detailed-options"},"table")," provides more detail."),(0,r.kt)("h3",{id:"configuring-components"},"Configuring Components"),(0,r.kt)("p",null,"An ",(0,r.kt)("em",{parentName:"p"},"ImageJob")," is made up of various sub-jobs, with one sub-job for each node.\nThese sub-jobs can be broken down further into three stages."),(0,r.kt)("ol",null,(0,r.kt)("li",{parentName:"ol"},"Collection (What is on the node?)"),(0,r.kt)("li",{parentName:"ol"},"Scanning (What images conform to the policy I've provided?)"),(0,r.kt)("li",{parentName:"ol"},"Removal (Remove images based on the results of the above)")),(0,r.kt)("p",null,"Of the above stages, only Removal is mandatory. The others can be disabled.\nFurthermore, manually triggered ",(0,r.kt)("em",{parentName:"p"},"ImageJobs")," will skip right to removal, even if\nEraser is configured to collect and scan. Collection and Scanning will only\ntake place when:"),(0,r.kt)("ol",null,(0,r.kt)("li",{parentName:"ol"},"The collector and/or scanner ",(0,r.kt)("inlineCode",{parentName:"li"},"components")," are enabled, AND"),(0,r.kt)("li",{parentName:"ol"},"The job was ",(0,r.kt)("em",{parentName:"li"},"not")," triggered manually by creating an ",(0,r.kt)("em",{parentName:"li"},"ImageList"),".")),(0,r.kt)("h3",{id:"swapping-out-components"},"Swapping out components"),(0,r.kt)("p",null,"The collector, scanner, and eraser components can all be swapped out. This\nenables you to build and host the images yourself. In addition, the scanner's\nbehavior can be completely tailored to your needs by swapping out the default\nimage with one of your own. To specify the images, use the\n",(0,r.kt)("inlineCode",{parentName:"p"},"components.<component>.image.repo")," and ",(0,r.kt)("inlineCode",{parentName:"p"},"components.<component>.image.tag"),",\nwhere ",(0,r.kt)("inlineCode",{parentName:"p"},"<component>")," is one of ",(0,r.kt)("inlineCode",{parentName:"p"},"collector"),", ",(0,r.kt)("inlineCode",{parentName:"p"},"scanner"),", or ",(0,r.kt)("inlineCode",{parentName:"p"},"eraser"),"."),(0,r.kt)("h2",{id:"universal-options"},"Universal Options"),(0,r.kt)("p",null,"The following portions of the configmap apply no matter how you spawn your\n",(0,r.kt)("em",{parentName:"p"},"ImageJob"),". The values provided below are the defaults. For more detail on\nthese options, see the ",(0,r.kt)("a",{parentName:"p",href:"#detailed-options"},"table"),"."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},'manager:\n  runtime: containerd\n  otlpEndpoint: "" # empty string disables OpenTelemetry\n  logLevel: info\n  profile:\n    enabled: false\n    port: 6060\n  imageJob:\n    successRatio: 1.0\n    cleanup:\n      delayOnSuccess: 0s\n      delayOnFailure: 24h\n  pullSecrets: [] # image pull secrets for collector/scanner/eraser\n  priorityClassName: "" # priority class name for collector/scanner/eraser\n  nodeFilter:\n    type: exclude # must be either exclude|include\n    selectors:\n      - eraser.sh/cleanup.filter\n      - kubernetes.io/os=windows\ncomponents:\n  eraser:\n    image:\n      repo: ghcr.io/azure/eraser\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 0\n    limit:\n      mem: 30Mi\n      cpu: 1000m\n')),(0,r.kt)("h2",{id:"component-options"},"Component Options"),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},"components:\n  collector:\n    enabled: true\n    image:\n      repo: ghcr.io/azure/collector\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 7m\n    limit:\n      mem: 500Mi\n      cpu: 0\n  scanner:\n    enabled: true\n    image:\n      repo: ghcr.io/azure/eraser-trivy-scanner\n      tag: v1.0.0\n    request:\n      mem: 500Mi\n      cpu: 1000m\n    limit:\n      mem: 2Gi\n      cpu: 0\n    config: |\n      # this is the schema for the provided 'trivy-scanner'. custom scanners\n      # will define their own configuration. see the below\n  eraser:\n    image:\n      repo: ghcr.io/azure/eraser\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 0\n    limit:\n      mem: 30Mi\n      cpu: 1000m\n")),(0,r.kt)("h2",{id:"scanner-options"},"Scanner Options"),(0,r.kt)("p",null,"These options can be provided to ",(0,r.kt)("inlineCode",{parentName:"p"},"components.scanner.config"),". They will be\npassed through  as a string to the scanner container and parsed there. If you\nwant to configure your own scanner, you must provide some way to parse this."),(0,r.kt)("p",null,"Below are the values recognized by the provided ",(0,r.kt)("inlineCode",{parentName:"p"},"eraser-trivy-scanner")," image.\nValues provided below are the defaults."),(0,r.kt)("pre",null,(0,r.kt)("code",{parentName:"pre",className:"language-yaml"},"cacheDir: /var/lib/trivy # The file path inside the container to store the cache\ndbRepo: ghcr.io/aquasecurity/trivy-db # The container registry from which to fetch the trivy database\ndeleteFailedImages: true # if true, remove images for which scanning fails, regardless of why it failed\nvulnerabilities:\n  ignoreUnfixed: true # consider the image compliant if there are no known fixes for the vulnerabilities found.\n  types: # a list of vulnerability types. for more info, see trivy's documentation.\n    - os\n    - library\n  securityChecks: # see trivy's documentation for more invormation\n    - vuln\n  severities: # in this case, only flag images with CRITICAL vulnerability for removal\n    - CRITICAL\ntimeout:\n  total: 23h # if scanning isn't completed before this much time elapses, abort the whole scan\n  perImage: 1h # if scanning a single image exceeds this time, scanning will be aborted\n")),(0,r.kt)("h2",{id:"detailed-options"},"Detailed Options"),(0,r.kt)("table",null,(0,r.kt)("thead",{parentName:"table"},(0,r.kt)("tr",{parentName:"thead"},(0,r.kt)("th",{parentName:"tr",align:null},"Option"),(0,r.kt)("th",{parentName:"tr",align:null},"Description"),(0,r.kt)("th",{parentName:"tr",align:null},"Default"))),(0,r.kt)("tbody",{parentName:"table"},(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.runtime"),(0,r.kt)("td",{parentName:"tr",align:null},"The runtime to use for the manager's containers. Must be one of containerd, crio, or dockershim. It is assumed that your nodes are all using the same runtime, and there is currently no way to configure multiple runtimes."),(0,r.kt)("td",{parentName:"tr",align:null},"containerd")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.otlpEndpoint"),(0,r.kt)("td",{parentName:"tr",align:null},"The endpoint to send OpenTelemetry data to. If empty, data will not be sent."),(0,r.kt)("td",{parentName:"tr",align:null},'""')),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.logLevel"),(0,r.kt)("td",{parentName:"tr",align:null},"The log level for the manager's containers. Must be one of debug, info, warn, error, dpanic, panic, or fatal."),(0,r.kt)("td",{parentName:"tr",align:null},"info")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.scheduling.repeatInterval"),(0,r.kt)("td",{parentName:"tr",align:null},"Use only when collector ando/or scanner are enabled. This is like a cron job, and will spawn an ",(0,r.kt)("em",{parentName:"td"},"ImageJob")," at the interval provided."),(0,r.kt)("td",{parentName:"tr",align:null},"24h")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.scheduling.beginImmediately"),(0,r.kt)("td",{parentName:"tr",align:null},"If set to true, the fist ",(0,r.kt)("em",{parentName:"td"},"ImageJob")," will run immediately. If false, the job will not be spawned until after the interval (above) has elapsed."),(0,r.kt)("td",{parentName:"tr",align:null},"true")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.profile.enabled"),(0,r.kt)("td",{parentName:"tr",align:null},"Whether to enable profiling for the manager's containers. This is for debugging with ",(0,r.kt)("inlineCode",{parentName:"td"},"go tool pprof"),"."),(0,r.kt)("td",{parentName:"tr",align:null},"false")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.profile.port"),(0,r.kt)("td",{parentName:"tr",align:null},"The port on which to expose the profiling endpoint."),(0,r.kt)("td",{parentName:"tr",align:null},"6060")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.imageJob.successRatio"),(0,r.kt)("td",{parentName:"tr",align:null},"The ratio of successful image jobs required before a cleanup is performed."),(0,r.kt)("td",{parentName:"tr",align:null},"1.0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.imageJob.cleanup.delayOnSuccess"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of time to wait after a successful image job before performing cleanup."),(0,r.kt)("td",{parentName:"tr",align:null},"0s")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.imageJob.cleanup.delayOnFailure"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of time to wait after a failed image job before performing cleanup."),(0,r.kt)("td",{parentName:"tr",align:null},"24h")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.pullSecrets"),(0,r.kt)("td",{parentName:"tr",align:null},"The image pull secrets to use for collector, scanner, and eraser containers."),(0,r.kt)("td",{parentName:"tr",align:null},"[]")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.priorityClassName"),(0,r.kt)("td",{parentName:"tr",align:null},"The priority class to use for collector, scanner, and eraser containers."),(0,r.kt)("td",{parentName:"tr",align:null},'""')),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.nodeFilter.type"),(0,r.kt)("td",{parentName:"tr",align:null},'The type of node filter to use. Must be either "exclude" or "include".'),(0,r.kt)("td",{parentName:"tr",align:null},"exclude")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"manager.nodeFilter.selectors"),(0,r.kt)("td",{parentName:"tr",align:null},"A list of selectors used to filter nodes."),(0,r.kt)("td",{parentName:"tr",align:null},"[]")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.enabled"),(0,r.kt)("td",{parentName:"tr",align:null},"Whether to enable the collector component."),(0,r.kt)("td",{parentName:"tr",align:null},"true")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.image.repo"),(0,r.kt)("td",{parentName:"tr",align:null},"The repository containing the collector image."),(0,r.kt)("td",{parentName:"tr",align:null},"ghcr.io/azure/collector")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.image.tag"),(0,r.kt)("td",{parentName:"tr",align:null},"The tag of the collector image."),(0,r.kt)("td",{parentName:"tr",align:null},"v1.0.0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.request.mem"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of memory to request for the collector container."),(0,r.kt)("td",{parentName:"tr",align:null},"25Mi")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.request.cpu"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of CPU to request for the collector container."),(0,r.kt)("td",{parentName:"tr",align:null},"7m")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.limit.mem"),(0,r.kt)("td",{parentName:"tr",align:null},"The maximum amount of memory the collector container is allowed to use."),(0,r.kt)("td",{parentName:"tr",align:null},"500Mi")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.collector.limit.cpu"),(0,r.kt)("td",{parentName:"tr",align:null},"The maximum amount of CPU the collector container is allowed to use."),(0,r.kt)("td",{parentName:"tr",align:null},"0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.enabled"),(0,r.kt)("td",{parentName:"tr",align:null},"Whether to enable the scanner component."),(0,r.kt)("td",{parentName:"tr",align:null},"true")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.image.repo"),(0,r.kt)("td",{parentName:"tr",align:null},"The repository containing the scanner image."),(0,r.kt)("td",{parentName:"tr",align:null},"ghcr.io/azure/eraser-trivy-scanner")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.image.tag"),(0,r.kt)("td",{parentName:"tr",align:null},"The tag of the scanner image."),(0,r.kt)("td",{parentName:"tr",align:null},"v1.0.0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.request.mem"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of memory to request for the scanner container."),(0,r.kt)("td",{parentName:"tr",align:null},"500Mi")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.request.cpu"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of CPU to request for the scanner container."),(0,r.kt)("td",{parentName:"tr",align:null},"1000m")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.limit.mem"),(0,r.kt)("td",{parentName:"tr",align:null},"The maximum amount of memory the scanner container is allowed to use."),(0,r.kt)("td",{parentName:"tr",align:null},"2Gi")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.limit.cpu"),(0,r.kt)("td",{parentName:"tr",align:null},"The maximum amount of CPU the scanner container is allowed to use."),(0,r.kt)("td",{parentName:"tr",align:null},"0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.scanner.config"),(0,r.kt)("td",{parentName:"tr",align:null},"The configuration to pass to the scanner container, as a YAML string."),(0,r.kt)("td",{parentName:"tr",align:null},"See YAML below")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.eraser.image.repo"),(0,r.kt)("td",{parentName:"tr",align:null},"The repository containing the eraser image."),(0,r.kt)("td",{parentName:"tr",align:null},"ghcr.io/azure/eraser")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.eraser.image.tag"),(0,r.kt)("td",{parentName:"tr",align:null},"The tag of the eraser image."),(0,r.kt)("td",{parentName:"tr",align:null},"v1.0.0")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.eraser.request.mem"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of memory to request for the eraser container."),(0,r.kt)("td",{parentName:"tr",align:null},"25Mi")),(0,r.kt)("tr",{parentName:"tbody"},(0,r.kt)("td",{parentName:"tr",align:null},"components.eraser.request.cpu"),(0,r.kt)("td",{parentName:"tr",align:null},"The amount of CPU to request for the eraser container."),(0,r.kt)("td",{parentName:"tr",align:null},"0")))))}u.isMDXComponent=!0}}]);