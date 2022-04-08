import * as core from "@actions/core"
import promise from 'bluebird'

let WAIT_FOR_FORK = 5


export async function forkRepo(octo:any,originRepo:any,ORIGIN_REPO:string,username:string) {
  let fork = null
  await originRepo.forks.create()
  var tryCounter = 0
  while (fork == null && tryCounter < WAIT_FOR_FORK) {
    core.info('--- waiting until repo is forked')
    promise.delay(tryCounter * 1000)
    fork = await octo.repos(username, ORIGIN_REPO).fetch()
    tryCounter++
  }
  if (fork == null) {
    core.info('--- could not fork the origin repo')
    return null
  }
  return fork
}

export async function createNewBranch(client:any,origin_owner:string,repo:string,owner:string, branchName:string) {
  var originCommits = await client.rest.repos.getBranch({owner:origin_owner,repo:repo,branch:"master"})
  var branch_hash = originCommits.data.commit.sha

  core.info('--- creating branch...')
  var branch = await client.rest.git.createRef({
    owner:owner,
    repo:repo,
    ref:'refs/heads/' + branchName,
    sha:branch_hash
  })
  return branch.data.object.sha
}

export async function commitChanges(client:any, owner:string, repo:string, branch:string, path:string, content:string, commitMessage:string, commitsha:string){
  // get tree sha
  const commitData = await client.rest.git.getCommit({owner,repo,commit_sha:commitsha})
  const treeSha = commitData.data.tree.sha

  // createBlobForFile 
  const blobData = await client.rest.git.createBlob({owner, repo, content, encoding: 'utf-8',})

  //  createNewTree 
  let tree: { path?: string; mode?: "100644" | "100755" | "040000" | "160000" | "120000"; type?: "blob" | "tree" | "commit"; sha?: string; content?: string; }[] = [{
    path: path,
    mode: `100644`,
    type: `blob`,
    sha: blobData.data.sha,
  }]
  const { data } = await client.rest.git.createTree({owner, repo, tree, base_tree: treeSha})

  // create new commit
  const NewCommit = (await client.rest.git.createCommit({owner, repo, message:commitMessage, tree: data.sha, parents: [commitsha]})).data

  // set branch to commit
  const branchRef = await client.rest.git.updateRef({owner, repo, ref: `heads/${branch}`, sha: NewCommit.sha})
  return branchRef.data.object.sha
}

export async function doPullRequest(originRepo:any,ORIGIN_BRANCH:string,branchName:string, username:string, title:string, prBody:string) {
  core.info('--- creating pull request...')
  const pullRequest = originRepo.pulls.create({
    title: title,
    body: prBody,
    head: username + ":" + branchName,
    base: ORIGIN_BRANCH
  })
  return {
    ok: true,
    created: true,
    pr: pullRequest,
  }  
}
