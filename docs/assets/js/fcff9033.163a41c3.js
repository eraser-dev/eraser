"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4497],{6979:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>c,contentTitle:()=>i,default:()=>h,frontMatter:()=>o,metadata:()=>a,toc:()=>l});var t=r(5893),s=r(1151);const o={title:"Customization"},i=void 0,a={id:"customization",title:"Customization",description:"Overview",source:"@site/docs/customization.md",sourceDirName:".",slug:"/customization",permalink:"/eraser/docs/next/customization",draft:!1,unlisted:!1,tags:[],version:"current",frontMatter:{title:"Customization"},sidebar:"sidebar",previous:{title:"Exclusion",permalink:"/eraser/docs/next/exclusion"},next:{title:"Metrics",permalink:"/eraser/docs/next/metrics"}},c={},l=[{value:"Overview",id:"overview",level:2},{value:"Key Concepts",id:"key-concepts",level:2},{value:"Basic architecture",id:"basic-architecture",level:3},{value:"Scheduling",id:"scheduling",level:3},{value:"Fault Tolerance",id:"fault-tolerance",level:3},{value:"Excluding Nodes",id:"excluding-nodes",level:3},{value:"Configuring Components",id:"configuring-components",level:3},{value:"Swapping out components",id:"swapping-out-components",level:3},{value:"Universal Options",id:"universal-options",level:2},{value:"Component Options",id:"component-options",level:2},{value:"Scanner Options",id:"scanner-options",level:2},{value:"Detailed Options",id:"detailed-options",level:2}];function d(e){const n={a:"a",code:"code",em:"em",h2:"h2",h3:"h3",li:"li",ol:"ol",p:"p",pre:"pre",table:"table",tbody:"tbody",td:"td",th:"th",thead:"thead",tr:"tr",...(0,s.a)(),...e.components};return(0,t.jsxs)(t.Fragment,{children:[(0,t.jsx)(n.h2,{id:"overview",children:"Overview"}),"\n",(0,t.jsx)(n.p,{children:"Eraser uses a configmap to configure its behavior. The configmap is part of the\ndeployment and it is not necessary to deploy it manually. Once deployed, the configmap\ncan be edited at any time:"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-bash",children:"kubectl edit configmap --namespace eraser-system eraser-manager-config\n"})}),"\n",(0,t.jsx)(n.p,{children:"If an eraser job is already running, the changes will not take effect until the job completes.\nThe configuration is in yaml."}),"\n",(0,t.jsx)(n.h2,{id:"key-concepts",children:"Key Concepts"}),"\n",(0,t.jsx)(n.h3,{id:"basic-architecture",children:"Basic architecture"}),"\n",(0,t.jsxs)(n.p,{children:["The ",(0,t.jsx)(n.em,{children:"manager"})," runs as a pod in your cluster and manages ",(0,t.jsx)(n.em,{children:"ImageJobs"}),". Think of\nan ",(0,t.jsx)(n.em,{children:"ImageJob"})," as a unit of work, performed on every node in your cluster. Each\nnode runs a sub-job. The goal of the ",(0,t.jsx)(n.em,{children:"ImageJob"})," is to assess the images on your\ncluster's nodes, and to remove the images you don't want. There are two stages:"]}),"\n",(0,t.jsxs)(n.ol,{children:["\n",(0,t.jsx)(n.li,{children:"Assessment"}),"\n",(0,t.jsx)(n.li,{children:"Removal."}),"\n"]}),"\n",(0,t.jsx)(n.h3,{id:"scheduling",children:"Scheduling"}),"\n",(0,t.jsxs)(n.p,{children:["An ",(0,t.jsx)(n.em,{children:"ImageJob"})," can either be created on-demand (see ",(0,t.jsx)(n.a,{href:"https://eraser-dev.github.io/eraser/docs/manual-removal",children:"Manual Removal"}),"),\nor they can be spawned on a timer like a cron job. On-demand jobs skip the\nassessment stage and get right down to the business of removing the images you\nspecified. The behavior of an on-demand job is quite different from that of\ntimed jobs."]}),"\n",(0,t.jsx)(n.h3,{id:"fault-tolerance",children:"Fault Tolerance"}),"\n",(0,t.jsxs)(n.p,{children:["Because an ",(0,t.jsx)(n.em,{children:"ImageJob"})," runs on every node in your cluster, and the conditions on\neach node may vary widely, some of the sub-jobs may fail. If you cannot\ntolerate any failure, set the ",(0,t.jsx)(n.code,{children:"manager.imageJob.successRatio"})," property to\n",(0,t.jsx)(n.code,{children:"1.0"}),". If 75% success sounds good to you, set it to ",(0,t.jsx)(n.code,{children:"0.75"}),". In that case, if\nfewer than 75% of the pods spawned by the ",(0,t.jsx)(n.em,{children:"ImageJob"})," report success, the job as\na whole will be marked as a failure."]}),"\n",(0,t.jsxs)(n.p,{children:["This is mainly to help diagnose error conditions. As such, you can set\n",(0,t.jsx)(n.code,{children:"manager.imageJob.cleanup.delayOnFailure"})," to a long value so that logs can be\ncaptured before the spawned pods are cleaned up."]}),"\n",(0,t.jsx)(n.h3,{id:"excluding-nodes",children:"Excluding Nodes"}),"\n",(0,t.jsxs)(n.p,{children:["For various reasons, you may want to prevent Eraser from scheduling pods on\ncertain nodes. To do so, the nodes can be given a special label. By default,\nthis label is ",(0,t.jsx)(n.code,{children:"eraser.sh/cleanup.filter"}),", but you can configure the behavior with\nthe options under ",(0,t.jsx)(n.code,{children:"manager.nodeFilter"}),". The ",(0,t.jsx)(n.a,{href:"#detailed-options",children:"table"})," provides more detail."]}),"\n",(0,t.jsx)(n.h3,{id:"configuring-components",children:"Configuring Components"}),"\n",(0,t.jsxs)(n.p,{children:["An ",(0,t.jsx)(n.em,{children:"ImageJob"})," is made up of various sub-jobs, with one sub-job for each node.\nThese sub-jobs can be broken down further into three stages."]}),"\n",(0,t.jsxs)(n.ol,{children:["\n",(0,t.jsx)(n.li,{children:"Collection (What is on the node?)"}),"\n",(0,t.jsx)(n.li,{children:"Scanning (What images conform to the policy I've provided?)"}),"\n",(0,t.jsx)(n.li,{children:"Removal (Remove images based on the results of the above)"}),"\n"]}),"\n",(0,t.jsxs)(n.p,{children:["Of the above stages, only Removal is mandatory. The others can be disabled.\nFurthermore, manually triggered ",(0,t.jsx)(n.em,{children:"ImageJobs"})," will skip right to removal, even if\nEraser is configured to collect and scan. Collection and Scanning will only\ntake place when:"]}),"\n",(0,t.jsxs)(n.ol,{children:["\n",(0,t.jsxs)(n.li,{children:["The collector and/or scanner ",(0,t.jsx)(n.code,{children:"components"})," are enabled, AND"]}),"\n",(0,t.jsxs)(n.li,{children:["The job was ",(0,t.jsx)(n.em,{children:"not"})," triggered manually by creating an ",(0,t.jsx)(n.em,{children:"ImageList"}),"."]}),"\n"]}),"\n",(0,t.jsx)(n.p,{children:"Disabling scanner will remove all non-running images by default."}),"\n",(0,t.jsx)(n.h3,{id:"swapping-out-components",children:"Swapping out components"}),"\n",(0,t.jsxs)(n.p,{children:["The collector, scanner, and remover components can all be swapped out. This\nenables you to build and host the images yourself. In addition, the scanner's\nbehavior can be completely tailored to your needs by swapping out the default\nimage with one of your own. To specify the images, use the\n",(0,t.jsx)(n.code,{children:"components.<component>.image.repo"})," and ",(0,t.jsx)(n.code,{children:"components.<component>.image.tag"}),",\nwhere ",(0,t.jsx)(n.code,{children:"<component>"})," is one of ",(0,t.jsx)(n.code,{children:"collector"}),", ",(0,t.jsx)(n.code,{children:"scanner"}),", or ",(0,t.jsx)(n.code,{children:"remover"}),"."]}),"\n",(0,t.jsx)(n.h2,{id:"universal-options",children:"Universal Options"}),"\n",(0,t.jsxs)(n.p,{children:["The following portions of the configmap apply no matter how you spawn your\n",(0,t.jsx)(n.em,{children:"ImageJob"}),". The values provided below are the defaults. For more detail on\nthese options, see the ",(0,t.jsx)(n.a,{href:"#detailed-options",children:"table"}),"."]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-yaml",children:'manager:\n  runtime:\n    name: containerd\n    address: unix:///run/containerd/containerd.sock\n  otlpEndpoint: "" # empty string disables OpenTelemetry\n  logLevel: info\n  profile:\n    enabled: false\n    port: 6060\n  imageJob:\n    successRatio: 1.0\n    cleanup:\n      delayOnSuccess: 0s\n      delayOnFailure: 24h\n  pullSecrets: [] # image pull secrets for collector/scanner/remover\n  priorityClassName: "" # priority class name for collector/scanner/remover\n  additionalPodLabels: {}\n  nodeFilter:\n    type: exclude # must be either exclude|include\n    selectors:\n      - eraser.sh/cleanup.filter\n      - kubernetes.io/os=windows\ncomponents:\n  remover:\n    image:\n      repo: ghcr.io/eraser-dev/remover\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 0\n    limit:\n      mem: 30Mi\n      cpu: 1000m\n'})}),"\n",(0,t.jsx)(n.h2,{id:"component-options",children:"Component Options"}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-yaml",children:"components:\n  collector:\n    enabled: true\n    image:\n      repo: ghcr.io/eraser-dev/collector\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 7m\n    limit:\n      mem: 500Mi\n      cpu: 0\n  scanner:\n    enabled: true\n    image:\n      repo: ghcr.io/eraser-dev/eraser-trivy-scanner\n      tag: v1.0.0\n    request:\n      mem: 500Mi\n      cpu: 1000m\n    limit:\n      mem: 2Gi\n      cpu: 0\n    config: |\n      # this is the schema for the provided 'trivy-scanner'. custom scanners\n      # will define their own configuration. see the below\n  remover:\n    image:\n      repo: ghcr.io/eraser-dev/remover\n      tag: v1.0.0\n    request:\n      mem: 25Mi\n      cpu: 0\n    limit:\n      mem: 30Mi\n      cpu: 1000m\n"})}),"\n",(0,t.jsx)(n.h2,{id:"scanner-options",children:"Scanner Options"}),"\n",(0,t.jsxs)(n.p,{children:["These options can be provided to ",(0,t.jsx)(n.code,{children:"components.scanner.config"}),". They will be\npassed through  as a string to the scanner container and parsed there. If you\nwant to configure your own scanner, you must provide some way to parse this."]}),"\n",(0,t.jsxs)(n.p,{children:["Below are the values recognized by the provided ",(0,t.jsx)(n.code,{children:"eraser-trivy-scanner"})," image.\nValues provided below are the defaults."]}),"\n",(0,t.jsx)(n.pre,{children:(0,t.jsx)(n.code,{className:"language-yaml",children:"cacheDir: /var/lib/trivy # The file path inside the container to store the cache\ndbRepo: ghcr.io/aquasecurity/trivy-db # The container registry from which to fetch the trivy database\ndeleteFailedImages: true # if true, remove images for which scanning fails, regardless of why it failed\ndeleteEOLImages: true # if true, remove images that have reached their end-of-life date\nvulnerabilities:\n  ignoreUnfixed: true # consider the image compliant if there are no known fixes for the vulnerabilities found.\n  types: # a list of vulnerability types. for more info, see trivy's documentation.\n    - os\n    - library\n  securityChecks: # see trivy's documentation for more information\n    - vuln\n  severities: # in this case, only flag images with CRITICAL vulnerability for removal\n    - CRITICAL\n  ignoredStatuses: # a list of trivy statuses to ignore. See https://aquasecurity.github.io/trivy/v0.44/docs/configuration/filtering/#by-status.\ntimeout:\n  total: 23h # if scanning isn't completed before this much time elapses, abort the whole scan\n  perImage: 1h # if scanning a single image exceeds this time, scanning will be aborted\n"})}),"\n",(0,t.jsx)(n.h2,{id:"detailed-options",children:"Detailed Options"}),"\n",(0,t.jsxs)(n.table,{children:[(0,t.jsx)(n.thead,{children:(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.th,{children:"Option"}),(0,t.jsx)(n.th,{children:"Description"}),(0,t.jsx)(n.th,{children:"Default"})]})}),(0,t.jsxs)(n.tbody,{children:[(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.runtime.name"}),(0,t.jsx)(n.td,{children:"The runtime to use for the manager's containers. Must be one of containerd, crio, or dockershim. It is assumed that your nodes are all using the same runtime, and there is currently no way to configure multiple runtimes."}),(0,t.jsx)(n.td,{children:"containerd"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.runtime.address"}),(0,t.jsx)(n.td,{children:"The runtime socket address to use for the containers. Can provide a custom address for containerd and dockershim runtimes, but not for crio due to Trivy restrictions."}),(0,t.jsx)(n.td,{children:"unix:///run/containerd/containerd.sock"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.otlpEndpoint"}),(0,t.jsx)(n.td,{children:"The endpoint to send OpenTelemetry data to. If empty, data will not be sent."}),(0,t.jsx)(n.td,{children:'""'})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.logLevel"}),(0,t.jsx)(n.td,{children:"The log level for the manager's containers. Must be one of debug, info, warn, error, dpanic, panic, or fatal."}),(0,t.jsx)(n.td,{children:"info"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.scheduling.repeatInterval"}),(0,t.jsxs)(n.td,{children:["Use only when collector ando/or scanner are enabled. This is like a cron job, and will spawn an ",(0,t.jsx)(n.em,{children:"ImageJob"})," at the interval provided."]}),(0,t.jsx)(n.td,{children:"24h"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.scheduling.beginImmediately"}),(0,t.jsxs)(n.td,{children:["If set to true, the fist ",(0,t.jsx)(n.em,{children:"ImageJob"})," will run immediately. If false, the job will not be spawned until after the interval (above) has elapsed."]}),(0,t.jsx)(n.td,{children:"true"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.profile.enabled"}),(0,t.jsxs)(n.td,{children:["Whether to enable profiling for the manager's containers. This is for debugging with ",(0,t.jsx)(n.code,{children:"go tool pprof"}),"."]}),(0,t.jsx)(n.td,{children:"false"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.profile.port"}),(0,t.jsx)(n.td,{children:"The port on which to expose the profiling endpoint."}),(0,t.jsx)(n.td,{children:"6060"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.imageJob.successRatio"}),(0,t.jsx)(n.td,{children:"The ratio of successful image jobs required before a cleanup is performed."}),(0,t.jsx)(n.td,{children:"1.0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.imageJob.cleanup.delayOnSuccess"}),(0,t.jsx)(n.td,{children:"The amount of time to wait after a successful image job before performing cleanup."}),(0,t.jsx)(n.td,{children:"0s"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.imageJob.cleanup.delayOnFailure"}),(0,t.jsx)(n.td,{children:"The amount of time to wait after a failed image job before performing cleanup."}),(0,t.jsx)(n.td,{children:"24h"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.pullSecrets"}),(0,t.jsx)(n.td,{children:"The image pull secrets to use for collector, scanner, and remover containers."}),(0,t.jsx)(n.td,{children:"[]"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.priorityClassName"}),(0,t.jsx)(n.td,{children:"The priority class to use for collector, scanner, and remover containers."}),(0,t.jsx)(n.td,{children:'""'})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.additionalPodLabels"}),(0,t.jsx)(n.td,{children:"Additional labels for all pods that the controller creates at runtime."}),(0,t.jsx)(n.td,{children:(0,t.jsx)(n.code,{children:"{}"})})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.nodeFilter.type"}),(0,t.jsx)(n.td,{children:'The type of node filter to use. Must be either "exclude" or "include".'}),(0,t.jsx)(n.td,{children:"exclude"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"manager.nodeFilter.selectors"}),(0,t.jsx)(n.td,{children:"A list of selectors used to filter nodes."}),(0,t.jsx)(n.td,{children:"[]"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.enabled"}),(0,t.jsx)(n.td,{children:"Whether to enable the collector component."}),(0,t.jsx)(n.td,{children:"true"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.image.repo"}),(0,t.jsx)(n.td,{children:"The repository containing the collector image."}),(0,t.jsx)(n.td,{children:"ghcr.io/eraser-dev/collector"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.image.tag"}),(0,t.jsx)(n.td,{children:"The tag of the collector image."}),(0,t.jsx)(n.td,{children:"v1.0.0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.request.mem"}),(0,t.jsx)(n.td,{children:"The amount of memory to request for the collector container."}),(0,t.jsx)(n.td,{children:"25Mi"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.request.cpu"}),(0,t.jsx)(n.td,{children:"The amount of CPU to request for the collector container."}),(0,t.jsx)(n.td,{children:"7m"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.limit.mem"}),(0,t.jsx)(n.td,{children:"The maximum amount of memory the collector container is allowed to use."}),(0,t.jsx)(n.td,{children:"500Mi"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.collector.limit.cpu"}),(0,t.jsx)(n.td,{children:"The maximum amount of CPU the collector container is allowed to use."}),(0,t.jsx)(n.td,{children:"0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.enabled"}),(0,t.jsx)(n.td,{children:"Whether to enable the scanner component."}),(0,t.jsx)(n.td,{children:"true"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.image.repo"}),(0,t.jsx)(n.td,{children:"The repository containing the scanner image."}),(0,t.jsx)(n.td,{children:"ghcr.io/eraser-dev/eraser-trivy-scanner"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.image.tag"}),(0,t.jsx)(n.td,{children:"The tag of the scanner image."}),(0,t.jsx)(n.td,{children:"v1.0.0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.request.mem"}),(0,t.jsx)(n.td,{children:"The amount of memory to request for the scanner container."}),(0,t.jsx)(n.td,{children:"500Mi"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.request.cpu"}),(0,t.jsx)(n.td,{children:"The amount of CPU to request for the scanner container."}),(0,t.jsx)(n.td,{children:"1000m"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.limit.mem"}),(0,t.jsx)(n.td,{children:"The maximum amount of memory the scanner container is allowed to use."}),(0,t.jsx)(n.td,{children:"2Gi"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.limit.cpu"}),(0,t.jsx)(n.td,{children:"The maximum amount of CPU the scanner container is allowed to use."}),(0,t.jsx)(n.td,{children:"0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.scanner.config"}),(0,t.jsx)(n.td,{children:"The configuration to pass to the scanner container, as a YAML string."}),(0,t.jsx)(n.td,{children:"See YAML below"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.remover.image.repo"}),(0,t.jsx)(n.td,{children:"The repository containing the remover image."}),(0,t.jsx)(n.td,{children:"ghcr.io/eraser-dev/remover"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.remover.image.tag"}),(0,t.jsx)(n.td,{children:"The tag of the remover image."}),(0,t.jsx)(n.td,{children:"v1.0.0"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.remover.request.mem"}),(0,t.jsx)(n.td,{children:"The amount of memory to request for the remover container."}),(0,t.jsx)(n.td,{children:"25Mi"})]}),(0,t.jsxs)(n.tr,{children:[(0,t.jsx)(n.td,{children:"components.remover.request.cpu"}),(0,t.jsx)(n.td,{children:"The amount of CPU to request for the remover container."}),(0,t.jsx)(n.td,{children:"0"})]})]})]})]})}function h(e={}){const{wrapper:n}={...(0,s.a)(),...e.components};return n?(0,t.jsx)(n,{...e,children:(0,t.jsx)(d,{...e})}):d(e)}},1151:(e,n,r)=>{r.d(n,{Z:()=>a,a:()=>i});var t=r(7294);const s={},o=t.createContext(s);function i(e){const n=t.useContext(o);return t.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:i(e.components),t.createElement(o.Provider,{value:n},e.children)}}}]);