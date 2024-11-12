"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[6508],{9628:(e,s,n)=>{n.r(s),n.d(s,{assets:()=>o,contentTitle:()=>l,default:()=>h,frontMatter:()=>i,metadata:()=>r,toc:()=>d});const r=JSON.parse('{"id":"release-management","title":"Release Management","description":"Overview","source":"@site/versioned_docs/version-v1.4.x/release-management.md","sourceDirName":".","slug":"/release-management","permalink":"/eraser/docs/release-management","draft":false,"unlisted":false,"tags":[],"version":"v1.4.x","frontMatter":{}}');var a=n(4848),t=n(8453);const i={},l="Release Management",o={},d=[{value:"Overview",id:"overview",level:2},{value:"Legend",id:"legend",level:2},{value:"Release Versioning",id:"release-versioning",level:2},{value:"Supported Releases",id:"supported-releases",level:2},{value:"Supported Kubernetes Versions",id:"supported-kubernetes-versions",level:2},{value:"Acknowledgement",id:"acknowledgement",level:2}];function c(e){const s={a:"a",em:"em",h1:"h1",h2:"h2",header:"header",li:"li",p:"p",strong:"strong",ul:"ul",...(0,t.R)(),...e.components};return(0,a.jsxs)(a.Fragment,{children:[(0,a.jsx)(s.header,{children:(0,a.jsx)(s.h1,{id:"release-management",children:"Release Management"})}),"\n",(0,a.jsx)(s.h2,{id:"overview",children:"Overview"}),"\n",(0,a.jsx)(s.p,{children:"This document describes Eraser project release management, which includes release versioning, supported releases, and supported upgrades."}),"\n",(0,a.jsx)(s.h2,{id:"legend",children:"Legend"}),"\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsxs)(s.li,{children:[(0,a.jsx)(s.strong,{children:"X.Y.Z"})," refers to the version (git tag) of Eraser that is released. This is the version of the Eraser images and the Chart version."]}),"\n",(0,a.jsxs)(s.li,{children:[(0,a.jsx)(s.strong,{children:"Breaking changes"})," refer to schema changes, flag changes, and behavior changes of Eraser that may require a clean installation during upgrade, and it may introduce changes that could break backward compatibility."]}),"\n",(0,a.jsxs)(s.li,{children:[(0,a.jsx)(s.strong,{children:"Milestone"})," should be designed to include feature sets to accommodate 2 months release cycles including test gates. GitHub's milestones are used by maintainers to manage each release. PRs and Issues for each release should be created as part of a corresponding milestone."]}),"\n",(0,a.jsxs)(s.li,{children:[(0,a.jsx)(s.strong,{children:"Patch releases"})," refer to applicable fixes, including security fixes, may be backported to support releases, depending on severity and feasibility."]}),"\n",(0,a.jsxs)(s.li,{children:[(0,a.jsx)(s.strong,{children:"Test gates"})," should include soak tests and upgrade tests from the last minor version."]}),"\n"]}),"\n",(0,a.jsx)(s.h2,{id:"release-versioning",children:"Release Versioning"}),"\n",(0,a.jsxs)(s.p,{children:["All releases will be of the form ",(0,a.jsx)(s.em,{children:"vX.Y.Z"})," where X is the major version, Y is the minor version and Z is the patch version. This project strictly follows semantic versioning."]}),"\n",(0,a.jsx)(s.p,{children:"The rest of the doc will cover the release process for the following kinds of releases:"}),"\n",(0,a.jsx)(s.p,{children:(0,a.jsx)(s.strong,{children:"Major Releases"})}),"\n",(0,a.jsx)(s.p,{children:"No plan to move to 2.0.0 unless there is a major design change like an incompatible API change in the project"}),"\n",(0,a.jsx)(s.p,{children:(0,a.jsx)(s.strong,{children:"Minor Releases"})}),"\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsxs)(s.li,{children:["X.Y.0-alpha.W, W >= 0 (Branch : main)","\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsx)(s.li,{children:"Released as needed before we cut a beta X.Y release"}),"\n",(0,a.jsx)(s.li,{children:"Alpha release, cut from master branch"}),"\n"]}),"\n"]}),"\n",(0,a.jsxs)(s.li,{children:["X.Y.0-beta.W, W >= 0 (Branch : main)","\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsx)(s.li,{children:"Released as needed before we cut a stable X.Y release"}),"\n",(0,a.jsx)(s.li,{children:"More stable than the alpha release to signal users to test things out"}),"\n",(0,a.jsx)(s.li,{children:"Beta release, cut from master branch"}),"\n"]}),"\n"]}),"\n",(0,a.jsxs)(s.li,{children:["X.Y.0-rc.W, W >= 0 (Branch : main)","\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsx)(s.li,{children:"Released as needed before we cut a stable X.Y release"}),"\n",(0,a.jsx)(s.li,{children:"soak for ~ 2 weeks before cutting a stable release"}),"\n",(0,a.jsx)(s.li,{children:"Release candidate release, cut from master branch"}),"\n"]}),"\n"]}),"\n",(0,a.jsxs)(s.li,{children:["X.Y.0 (Branch: main)","\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsx)(s.li,{children:"Released as needed"}),"\n",(0,a.jsx)(s.li,{children:"Stable release, cut from master when X.Y milestone is complete"}),"\n"]}),"\n"]}),"\n"]}),"\n",(0,a.jsx)(s.p,{children:(0,a.jsx)(s.strong,{children:"Patch Releases"})}),"\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsxs)(s.li,{children:["Patch Releases X.Y.Z, Z > 0 (Branch: release-X.Y, only cut when a patch is needed)","\n",(0,a.jsxs)(s.ul,{children:["\n",(0,a.jsx)(s.li,{children:"No breaking changes"}),"\n",(0,a.jsx)(s.li,{children:"Applicable fixes, including security fixes, may be cherry-picked from master into the latest supported minor release-X.Y branches."}),"\n",(0,a.jsx)(s.li,{children:"Patch release, cut from a release-X.Y branch"}),"\n"]}),"\n"]}),"\n"]}),"\n",(0,a.jsx)(s.h2,{id:"supported-releases",children:"Supported Releases"}),"\n",(0,a.jsx)(s.p,{children:"Applicable fixes, including security fixes, may be cherry-picked into the release branch, depending on severity and feasibility. Patch releases are cut from that branch as needed."}),"\n",(0,a.jsx)(s.p,{children:"We expect users to stay reasonably up-to-date with the versions of Eraser they use in production, but understand that it may take time to upgrade. We expect users to be running approximately the latest patch release of a given minor release and encourage users to upgrade as soon as possible."}),"\n",(0,a.jsx)(s.p,{children:'We expect to "support" n (current) and n-1 major.minor releases. "Support" means we expect users to be running that version in production. For example, when v1.2.0 comes out, v1.0.x will no longer be supported for patches, and we encourage users to upgrade to a supported version as soon as possible.'}),"\n",(0,a.jsx)(s.h2,{id:"supported-kubernetes-versions",children:"Supported Kubernetes Versions"}),"\n",(0,a.jsxs)(s.p,{children:["Eraser is assumed to be compatible with the ",(0,a.jsx)(s.a,{href:"https://kubernetes.io/releases/patch-releases/#detailed-release-history-for-active-branches",children:"current Kubernetes Supported Versions"})," per ",(0,a.jsx)(s.a,{href:"https://kubernetes.io/releases/version-skew-policy/",children:"Kubernetes Supported Versions policy"}),"."]}),"\n",(0,a.jsxs)(s.p,{children:["For example, if Eraser ",(0,a.jsx)(s.em,{children:"supported"})," versions are v1.2 and v1.1, and Kubernetes ",(0,a.jsx)(s.em,{children:"supported"})," versions are v1.22, v1.23, v1.24, then all supported Eraser versions (v1.2, v1.1) are assumed to be compatible with all supported Kubernetes versions (v1.22, v1.23, v1.24). If Kubernetes v1.25 is released later, then Eraser v1.2 and v1.1 will be assumed to be compatible with v1.25 if those Eraser versions are still supported at that time."]}),"\n",(0,a.jsx)(s.p,{children:"If you choose to use Eraser with a version of Kubernetes that it does not support, you are using it at your own risk."}),"\n",(0,a.jsx)(s.h2,{id:"acknowledgement",children:"Acknowledgement"}),"\n",(0,a.jsx)(s.p,{children:"This document builds on the ideas and implementations of release processes from projects like Kubernetes and Helm."})]})}function h(e={}){const{wrapper:s}={...(0,t.R)(),...e.components};return s?(0,a.jsx)(s,{...e,children:(0,a.jsx)(c,{...e})}):c(e)}},8453:(e,s,n)=>{n.d(s,{R:()=>i,x:()=>l});var r=n(6540);const a={},t=r.createContext(a);function i(e){const s=r.useContext(t);return r.useMemo((function(){return"function"==typeof e?e(s):{...s,...e}}),[s,e])}function l(e){let s;return s=e.disableParentContext?"function"==typeof e.components?e.components(a):e.components||a:i(e.components),r.createElement(t.Provider,{value:s},e.children)}}}]);