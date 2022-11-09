import { Octokit } from "@octokit/core";
import { Api } from "@octokit/plugin-rest-endpoint-methods/dist-types/types";
import * as core from "@actions/core";
import { exit } from "process";

export async function handleKBIssue(
  octokit: Octokit & Api,
  owner,
  repo,
  issue: { title: string; number: number }
) {
  const storage_issue = 1380;
  const comment_id = 1308209074;
  let comment = await prepareComment(octokit, owner, repo, issue);
  core.info(`Analysis For ${issue}:\n ${comment}`);

  let resp = await octokit.rest.issues.getComment({
    owner: owner,
    repo: repo,
    comment_id: comment_id,
  });

  if (resp.status == 200) {
    let old_body = resp.data.body;
    let new_body = old_body + comment;

    let resp2 = await octokit.rest.issues.updateComment({
      owner: owner,
      repo: repo,
      comment_id: comment_id,
      body: new_body,
    });
    if (resp2.status !== 200) {
      core.info(`[X] Unable to add: ${issue.number} in the tracking comment`);
    } else {
      core.info(`[!] Added ${issue.title} in tracking comment.`);
      let resp3 = await octokit.rest.issues.update({
        owner: owner,
        repo: repo,
        issue_number: issue.number,
        state: "closed",
      });
      if (resp3.status === 200) {
        core.info(`[!] Closed Issue ${issue.number}`);
      } else {
        core.info(`[X] Unable to close issue ${issue.number}`);
      }
    }
    exit(0);
  }
  core.info(`[X] Unable to handle: ${issue.title} `);
  exit(0);
}

function createIssueCommentBody(data: { title: string; body: string }) {
  let output = [];
  output.push(`\n- [ ] ${data.title.substring(5)}`);
  let new_body = data.body.split("\n");
  output.push("  <details>");
  output.push("  <summary>Analysis</summary>\n");
  for (let line of new_body) {
    output.push(`  ${line}`);
  }
  output.push("  </details>");
  return output.join("\n");
}

async function prepareComment(
  client: Octokit & Api,
  owner,
  repo,
  issue: { title: string; number: number }
) {
  let resp = await client.rest.issues.listComments({
    owner: owner,
    repo: repo,
    issue_number: issue.number,
  });

  if (resp.status === 200) {
    if (resp.data.length > 0) {
      let body = resp.data[0].body;
      return createIssueCommentBody({ title: issue.title, body: body });
    } else {
      return createIssueCommentBody({
        title: issue.title,
        body: "no analysis found",
      });
    }
  }

  return createIssueCommentBody({
    title: issue.title,
    body: "unable to fetch analysis",
  });
}
