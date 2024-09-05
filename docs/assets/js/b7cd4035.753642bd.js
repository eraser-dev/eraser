"use strict";(self.webpackChunkwebsite=self.webpackChunkwebsite||[]).push([[4695],{6737:(e,n,r)=>{r.r(n),r.d(n,{assets:()=>o,contentTitle:()=>l,default:()=>d,frontMatter:()=>t,metadata:()=>a,toc:()=>c});var s=r(4848),i=r(8453);const t={title:"Releasing"},l=void 0,a={id:"releasing",title:"Releasing",description:"Create Release Pull Request",source:"@site/versioned_docs/version-v1.3.x/releasing.md",sourceDirName:".",slug:"/releasing",permalink:"/eraser/docs/v1.3.x/releasing",draft:!1,unlisted:!1,tags:[],version:"v1.3.x",frontMatter:{title:"Releasing"},sidebar:"sidebar",previous:{title:"Setup",permalink:"/eraser/docs/v1.3.x/setup"},next:{title:"Custom Scanner",permalink:"/eraser/docs/v1.3.x/custom-scanner"}},o={},c=[{value:"Create Release Pull Request",id:"create-release-pull-request",level:2},{value:"Publishing",id:"publishing",level:2},{value:"Notifying",id:"notifying",level:2}];function h(e){const n={a:"a",code:"code",h1:"h1",h2:"h2",li:"li",ol:"ol",p:"p",pre:"pre",...(0,i.R)(),...e.components};return(0,s.jsxs)(s.Fragment,{children:[(0,s.jsx)(n.h2,{id:"create-release-pull-request",children:"Create Release Pull Request"}),"\n",(0,s.jsxs)(n.ol,{children:["\n",(0,s.jsxs)(n.li,{children:["Go to ",(0,s.jsx)(n.code,{children:"create_release_pull_request"})," workflow under actions."]}),"\n",(0,s.jsx)(n.li,{children:"Select run workflow, and use the workflow from your branch."}),"\n",(0,s.jsx)(n.li,{children:"Input release version with the semantic version identifying the release."}),"\n",(0,s.jsx)(n.li,{children:"Click run workflow and review the PR created by github-actions."}),"\n"]}),"\n",(0,s.jsx)(n.h1,{id:"releasing",children:"Releasing"}),"\n",(0,s.jsxs)(n.ol,{start:"5",children:["\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:["Once the PR is merged to ",(0,s.jsx)(n.code,{children:"main"}),", tag that commit with release version and push tags to remote repository."]}),"\n",(0,s.jsx)(n.pre,{children:(0,s.jsx)(n.code,{children:"git checkout <BRANCH NAME>\ngit pull origin <BRANCH NAME>\ngit tag -a <NEW VERSION> -m '<NEW VERSION>'\ngit push origin <NEW VERSION>\n"})}),"\n"]}),"\n",(0,s.jsxs)(n.li,{children:["\n",(0,s.jsxs)(n.p,{children:["Pushing the release tag will trigger GitHub Actions to trigger ",(0,s.jsx)(n.code,{children:"release"})," job.\nThis will build the ",(0,s.jsx)(n.code,{children:"ghcr.io/eraser-dev/remover"}),", ",(0,s.jsx)(n.code,{children:"ghcr.io/eraser-dev/eraser-manager"}),", ",(0,s.jsx)(n.code,{children:"ghcr.io/eraser-dev/collector"}),", and ",(0,s.jsx)(n.code,{children:"ghcr.io/eraser-dev/eraser-trivy-scanner"})," images automatically, then publish the new release tag."]}),"\n"]}),"\n"]}),"\n",(0,s.jsx)(n.h2,{id:"publishing",children:"Publishing"}),"\n",(0,s.jsxs)(n.ol,{children:["\n",(0,s.jsxs)(n.li,{children:["GitHub Action will create a new release, review and edit it at ",(0,s.jsx)(n.a,{href:"https://github.com/eraser-dev/eraser/releases",children:"https://github.com/eraser-dev/eraser/releases"})]}),"\n"]}),"\n",(0,s.jsx)(n.h2,{id:"notifying",children:"Notifying"}),"\n",(0,s.jsxs)(n.ol,{children:["\n",(0,s.jsxs)(n.li,{children:["Send an email to the ",(0,s.jsx)(n.a,{href:"https://groups.google.com/g/eraser-dev",children:"Eraser mailing list"})," announcing the release, with links to GitHub."]}),"\n",(0,s.jsxs)(n.li,{children:["Post a message on the ",(0,s.jsx)(n.a,{href:"https://kubernetes.slack.com/archives/C03Q8KV8YQ4",children:"Eraser Slack channel"})," with the same information."]}),"\n"]})]})}function d(e={}){const{wrapper:n}={...(0,i.R)(),...e.components};return n?(0,s.jsx)(n,{...e,children:(0,s.jsx)(h,{...e})}):h(e)}},8453:(e,n,r)=>{r.d(n,{R:()=>l,x:()=>a});var s=r(6540);const i={},t=s.createContext(i);function l(e){const n=s.useContext(t);return s.useMemo((function(){return"function"==typeof e?e(n):{...n,...e}}),[n,e])}function a(e){let n;return n=e.disableParentContext?"function"==typeof e.components?e.components(i):e.components||i:l(e.components),s.createElement(t.Provider,{value:n},e.children)}}}]);