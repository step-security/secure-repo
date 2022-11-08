import { Octokit } from "@octokit/core";
import { Api } from "@octokit/plugin-rest-endpoint-methods/dist-types/types";
import * as core from "@actions/core"


export async function handleKBIssue(
  octokit: Octokit & Api,
  owner,
  repo,
  issue: {title:string, number:number}
) {
  let comment = await prepareComment(octokit, owner, repo, issue);
  core.info(`Analysis For ${issue}:\n ${comment}`)
}



function createIssueCommentBody(data: {title:string, body:string}){
    let output = []
    output.push(`- [ ] ${data.title}`)
    let new_body = data.body.split("\n")
    output.push("  <details>")
    output.push("  <summary>Analysis</summary")
    for(let line of new_body){
        output.push(`  ${line}`)
    }
    output.push("  </details>")
    return output.join("\n")

}

async function prepareComment(client: Octokit & Api, owner, repo, issue: {title:string, number:number}) {
  let resp = await client.rest.issues.listComments({
    owner: owner,
    repo: repo,
    issue_number: issue.number,
  });

  if(resp.status === 200){
    if(resp.data.length > 0){

        let body = resp.data[0].body
        return createIssueCommentBody({title:issue.title, body:body});
        
    }
  }
  
  return "not found"

}

