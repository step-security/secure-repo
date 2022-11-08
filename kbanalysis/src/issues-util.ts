import { Octokit } from "@octokit/core";
import { Api } from "@octokit/plugin-rest-endpoint-methods/dist-types/types";
import * as core from "@actions/core"


export async function handleKBIssue(
  octokit: Octokit & Api,
  owner,
  repo,
  issue: number
) {
  let analysis = await getAnalysis(octokit, owner, repo, issue);
  core.info(`Analysis For ${issue}:\n ${analysis}`)
}

async function getAnalysis(client: Octokit & Api, owner, repo, issue: number) {
  let resp = await client.rest.issues.listComments({
    owner: owner,
    repo: repo,
    issue_number: issue,
  });

  if(resp.status === 200){
    if(resp.data.length > 0){
        return resp.data[0].body
    }
  }
  
  return "not found"

}
