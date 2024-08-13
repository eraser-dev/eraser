"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[5581],{6366:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>c,contentTitle:()=>d,default:()=>a,frontMatter:()=>t,metadata:()=>l,toc:()=>h});var i=r(5893),s=r(1151);const t={title:"Setup"},d="Development Setup",l={id:"setup",title:"Setup",description:"This document describes the steps to get started with development.",source:"@site/docs/setup.md",sourceDirName:".",slug:"/setup",permalink:"/eraser/docs/next/setup",draft:!1,unlisted:!1,tags:[],version:"current",frontMatter:{title:"Setup"},sidebar:"sidebar",previous:{title:"Metrics",permalink:"/eraser/docs/next/metrics"},next:{title:"Releasing",permalink:"/eraser/docs/next/releasing"}},c={},h=[{value:"Local Setup",id:"local-setup",level:2},{value:"Prerequisites:",id:"prerequisites",level:3},{value:"Get things running",id:"get-things-running",level:3},{value:"Making changes",id:"making-changes",level:3},{value:"Development Reference",id:"development-reference",level:2},{value:"Common Configuration",id:"common-configuration",level:3},{value:"Linting",id:"linting",level:3},{value:"Development",id:"development",level:3},{value:"Build",id:"build",level:3},{value:"Deployment",id:"deployment",level:3},{value:"Release",id:"release",level:3}];function o(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",h3:"h3",header:"header",li:"li",p:"p",pre:"pre",table:"table",tbody:"tbody",td:"td",th:"th",thead:"thead",tr:"tr",ul:"ul",...(0,s.a)(),...e.components};return(0,i.jsxs)(i.Fragment,{children:[(0,i.jsx)(n.header,{children:(0,i.jsx)(n.h1,{id:"development-setup",children:"Development Setup"})}),"\n",(0,i.jsxs)(n.p,{children:["This document describes the steps to get started with development.\nYou can either utilize ",(0,i.jsx)(n.a,{href:"https://docs.github.com/en/codespaces/overview",children:"Codespaces"})," or setup a local environment."]}),"\n",(0,i.jsx)(n.h2,{id:"local-setup",children:"Local Setup"}),"\n",(0,i.jsx)(n.h3,{id:"prerequisites",children:"Prerequisites:"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:[(0,i.jsx)(n.a,{href:"https://go.dev/",children:"go"})," with version 1.17 or later."]}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"https://docs.docker.com/get-docker/",children:"docker"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.a,{href:"https://kind.sigs.k8s.io/",children:"kind"})}),"\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make"})}),"\n"]}),"\n",(0,i.jsx)(n.h3,{id:"get-things-running",children:"Get things running"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsxs)(n.li,{children:["\n",(0,i.jsxs)(n.p,{children:["Get dependencies with ",(0,i.jsx)(n.code,{children:"go get"})]}),"\n"]}),"\n",(0,i.jsxs)(n.li,{children:["\n",(0,i.jsxs)(n.p,{children:["This project uses ",(0,i.jsx)(n.code,{children:"make"}),". You can utilize ",(0,i.jsx)(n.code,{children:"make help"})," to see available targets. For local deployment make targets help to build, test and deploy."]}),"\n"]}),"\n"]}),"\n",(0,i.jsx)(n.h3,{id:"making-changes",children:"Making changes"}),"\n",(0,i.jsxs)(n.p,{children:["Please refer to ",(0,i.jsx)(n.a,{href:"#development-reference",children:"Development Reference"})," for more details on the specific commands."]}),"\n",(0,i.jsx)(n.p,{children:"To test your changes on a cluster:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-bash",children:"# generate necessary api files (optional - only needed if changes to api folder).\nmake generate\n\n# build applicable images\nmake docker-build-manager MANAGER_IMG=eraser-manager:dev\nmake docker-build-remover REMOVER_IMG=remover:dev\nmake docker-build-collector COLLECTOR_IMG=collector:dev\nmake docker-build-trivy-scanner TRIVY_SCANNER_IMG=eraser-trivy-scanner:dev\n\n# make sure updated image is present on cluster (e.g., see kind example below)\nkind load docker-image \\\n        eraser-manager:dev \\\n        eraser-trivy-scanner:dev \\\n        remover:dev \\\n        collector:dev\n\nmake manifests\nmake deploy\n\n# to remove the deployment\nmake undeploy\n"})}),"\n",(0,i.jsx)(n.p,{children:"To test your changes to manager locally:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{className:"language-bash",children:"make run\n"})}),"\n",(0,i.jsx)(n.p,{children:"Example Output:"}),"\n",(0,i.jsx)(n.pre,{children:(0,i.jsx)(n.code,{children:'you@local:~/eraser$ make run\ndocker build . \\\n        -t eraser-tooling \\\n        -f build/tooling/Dockerfile\n[+] Building 7.8s (8/8) FINISHED\n => => naming to docker.io/library/eraser-tooling                           0.0s\ndocker run -v /home/eraser/config:/config -w /config/manager \\\n        registry.k8s.io/kustomize/kustomize:v3.8.9 edit set image controller=eraser-manager:dev\ndocker run -v /home/eraser:/eraser eraser-tooling controller-gen \\\n        crd \\\n        rbac:roleName=manager-role \\\n        webhook \\\n        paths="./..." \\\n        output:crd:artifacts:config=config/crd/bases\nrm -rf manifest_staging\nmkdir -p manifest_staging/deploy\ndocker run --rm -v /home/eraser:/eraser \\\n        registry.k8s.io/kustomize/kustomize:v3.8.9 build \\\n        /eraser/config/default -o /eraser/manifest_staging/deploy/eraser.yaml\ndocker run -v /home/eraser:/eraser eraser-tooling controller-gen object:headerFile="hack/boilerplate.go.txt" paths="./..."\ngo fmt ./...\ngo vet ./...\ngo run ./main.go\n{"level":"info","ts":1652985685.1663408,"logger":"controller-runtime.metrics","msg":"Metrics server is starting to listen","addr":":8080"}\n...\n'})}),"\n",(0,i.jsx)(n.h2,{id:"development-reference",children:"Development Reference"}),"\n",(0,i.jsxs)(n.p,{children:["Eraser is using tooling from ",(0,i.jsx)(n.a,{href:"https://github.com/kubernetes-sigs/kubebuilder",children:"kubebuilder"}),". For Eraser this tooling is containerized into the ",(0,i.jsx)(n.code,{children:"eraser-tooling"})," image. The ",(0,i.jsx)(n.code,{children:"make"})," targets can use this tooling and build the image when necessary."]}),"\n",(0,i.jsx)(n.p,{children:"You can override the default configuration using environment variables. Below you can find a reference of targets and configuration options."}),"\n",(0,i.jsx)(n.h3,{id:"common-configuration",children:"Common Configuration"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"VERSION"}),(0,i.jsx)(n.td,{children:"Specifies the version (i.e., the image tag) of eraser to be used."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Defines the image url for the Eraser manager. Used for tagging, pulling and pushing the image"})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"REMOVER_IMG"}),(0,i.jsx)(n.td,{children:"Defines the image url for the Eraser. Used for tagging, pulling and pushing the image"})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"COLLECTOR_IMG"}),(0,i.jsx)(n.td,{children:"Defines the image url for the Collector. Used for tagging, pulling and pushing the image"})]})]})]}),"\n",(0,i.jsx)(n.h3,{id:"linting",children:"Linting"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make lint"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Lints the go code."}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"GOLANGCI_LINT"}),(0,i.jsx)(n.td,{children:"Specifies the go linting binary to be used for linting."})]})})]}),"\n",(0,i.jsx)(n.h3,{id:"development",children:"Development"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make generate"})}),"\n"]}),"\n",(0,i.jsxs)(n.p,{children:["Generates necessary files for the k8s api stored under ",(0,i.jsx)(n.code,{children:"api/v1alpha1/zz_generated.deepcopy.go"}),". See the ",(0,i.jsx)(n.a,{href:"https://book.kubebuilder.io/cronjob-tutorial/other-api-files.html",children:"kubebuilder docs"})," for details."]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make manifests"})}),"\n"]}),"\n",(0,i.jsxs)(n.p,{children:["Generates the eraser deployment yaml files under ",(0,i.jsx)(n.code,{children:"manifest_staging/deploy"}),"."]}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"REMOVER_IMG"}),(0,i.jsx)(n.td,{children:"Defines the image url for the Eraser."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Defines the image url for the Eraser manager."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUSTOMIZE_VERSION"}),(0,i.jsx)(n.td,{children:"Define Kustomize version for generating manifests."})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make test"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Runs the unit tests for the eraser project."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"ENVTEST"}),(0,i.jsx)(n.td,{children:"Specifies the envtest setup binary."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"ENVTEST_K8S_VERSION"}),(0,i.jsx)(n.td,{children:"Specifies the Kubernetes version for envtest setup command."})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make e2e-test"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Runs e2e tests on a cluster."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"REMOVER_IMG"}),(0,i.jsx)(n.td,{children:"Eraser image to be used for e2e test."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Eraser manager image to be used for e2e test."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUBERNETES_VERSION"}),(0,i.jsx)(n.td,{children:"Kubernetes version for e2e test."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"TEST_COUNT"}),(0,i.jsxs)(n.td,{children:["Sets repetition for test. Please refer to ",(0,i.jsx)(n.a,{href:"https://pkg.go.dev/cmd/go#hdr-Testing_flags",children:"go docs"})," for details."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"TIMEOUT"}),(0,i.jsxs)(n.td,{children:["Sets timeout for test. Please refer to ",(0,i.jsx)(n.a,{href:"https://pkg.go.dev/cmd/go#hdr-Testing_flags",children:"go docs"})," for details."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"TESTFLAGS"}),(0,i.jsx)(n.td,{children:"Sets additional test flags"})]})]})]}),"\n",(0,i.jsx)(n.h3,{id:"build",children:"Build"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make build"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the eraser manager binaries."}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make run"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Runs the eraser manager on your local machine."}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-build-manager"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for the eraser manager."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_FROM"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-from flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_TO"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-to flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"PLATFORM"}),(0,i.jsxs)(n.td,{children:["Sets the target platform for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#platform",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"OUTPUT_TYPE"}),(0,i.jsxs)(n.td,{children:["Sets the output for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#output",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-push-manager"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for the eraser manager."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-build-remover"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for eraser remover."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_FROM"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-from flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_TO"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-to flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"PLATFORM"}),(0,i.jsxs)(n.td,{children:["Sets the target platform for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#platform",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"OUTPUT_TYPE"}),(0,i.jsxs)(n.td,{children:["Sets the output for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#output",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"REMOVER_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-push-remover"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for the eraser remover."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"REMOVER_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-build-collector"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for the eraser collector."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_FROM"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-from flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-from",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"CACHE_TO"}),(0,i.jsxs)(n.td,{children:["Sets the target of the buildx --cache-to flag ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#cache-to",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"PLATFORM"}),(0,i.jsxs)(n.td,{children:["Sets the target platform for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#platform",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"OUTPUT_TYPE"}),(0,i.jsxs)(n.td,{children:["Sets the output for buildx ",(0,i.jsx)(n.a,{href:"https://docs.docker.com/engine/reference/commandline/buildx_build/#output",children:"see buildx reference"}),"."]})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"COLLECTOR_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make docker-push-collector"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Builds the docker image for the eraser collector."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"COLLECTOR_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the target repository, image name and tag for building image."})]})})]}),"\n",(0,i.jsx)(n.h3,{id:"deployment",children:"Deployment"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make install"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Install CRDs into the K8s cluster specified in ~/.kube/config."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUSTOMIZE_VERSION"}),(0,i.jsx)(n.td,{children:"Kustomize version used to generate k8s resources for deployment."})]})})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make uninstall"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Uninstall CRDs from the K8s cluster specified in ~/.kube/config."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUSTOMIZE_VERSION"}),(0,i.jsx)(n.td,{children:"Kustomize version used to generate k8s resources for deployment."})]})})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make deploy"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Deploys eraser to the cluster specified in ~/.kube/config."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsxs)(n.tbody,{children:[(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUSTOMIZE_VERSION"}),(0,i.jsx)(n.td,{children:"Kustomize version used to generate k8s resources for deployment."})]}),(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"MANAGER_IMG"}),(0,i.jsx)(n.td,{children:"Specifies the eraser manager image version to be used for deployment"})]})]})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make undeploy"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Undeploy controller from the K8s cluster specified in ~/.kube/config."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"KUSTOMIZE_VERSION"}),(0,i.jsx)(n.td,{children:"Kustomize version used to generate k8s resources that need to be removed."})]})})]}),"\n",(0,i.jsx)(n.h3,{id:"release",children:"Release"}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make release-manifest"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Generates k8s manifests files for a release."}),"\n",(0,i.jsx)(n.p,{children:"Configuration Options:"}),"\n",(0,i.jsxs)(n.table,{children:[(0,i.jsx)(n.thead,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.th,{children:"Environment Variable"}),(0,i.jsx)(n.th,{children:"Description"})]})}),(0,i.jsx)(n.tbody,{children:(0,i.jsxs)(n.tr,{children:[(0,i.jsx)(n.td,{children:"NEWVERSION"}),(0,i.jsx)(n.td,{children:"Sets the new version in the Makefile"})]})})]}),"\n",(0,i.jsxs)(n.ul,{children:["\n",(0,i.jsx)(n.li,{children:(0,i.jsx)(n.code,{children:"make promote-staging-manifest"})}),"\n"]}),"\n",(0,i.jsx)(n.p,{children:"Promotes the k8s deployment yaml files to release."})]})}function a(e={}){const{wrapper:n}={...(0,s.a)(),...e.components};return n?(0,i.jsx)(n,{...e,children:(0,i.jsx)(o,{...e})}):o(e)}},1151:(e,n,r)=>{r.d(n,{Z:()=>l,a:()=>d});var i=r(7294);const s={},t=i.createContext(s);function d(e){const n=i.useContext(t);return i.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function l(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(s):e.components||s:d(e.components),i.createElement(t.Provider,{value:n},e.children)}}}]);