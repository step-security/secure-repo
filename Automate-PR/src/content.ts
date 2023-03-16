export const prBody = `GitHub asks users to define workflow permissions, see https://github.blog/changelog/2021-04-20-github-actions-control-permissions-for-github_token/ and https://docs.github.com/en/actions/security-guides/automatic-token-authentication#modifying-the-permissions-for-the-github_token for securing GitHub workflows against supply-chain attacks.

StepSecurity is working on securing GitHub workflows and [OSSF Scorecards](https://github.com/ossf/scorecard) recommends using StepSecurity's secure-repo online tool [app.stepsecurity.io](https://github.com/cosmos/cosmos-sdk/pull/app.stepsecurity.io) to improve the security of GitHub workflows.

This repository has a Scorecards score of 4.5/10 with 10 being the most secure. The \`Token-Permissions\` category has a score of 0/10.

This file was fixed automatically using the open-source tool https://github.com/step-security/secure-repo. If you like the change, and merge it, please consider starring the repo. `
  
export const titlePR = "fix: permissions for "

export function get_pr_update(owner:string,repository:string,path:string,username:string,workflow:string){
let pr_update = `Details of Secured workflow
\`\`\`yml
    name: ${owner}
    repo: ${repository}
    path: ${path}
\`\`\`

links:
repo: https://github.com/${owner}/${repository}
fork: https://github.com/${username}/${repository}

> Secured Workflow
\`\`\`yml
${workflow}
\`\`\`
`
return pr_update
}