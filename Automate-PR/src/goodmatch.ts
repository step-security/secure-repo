import * as core from "@actions/core"
import { components } from '@octokit/openapi-types'

let CURR_PAGE = 1
let CURR_MATCH = 6

export async function get_details(client:any,issue_id:number, owner:string, repo:string){
  const resp=await client.rest.issues.get({issue_number: Number(issue_id ), owner: owner, repo:repo})
  const body:string=resp.data.body
  const body_content=body.split("\n")
  if(body_content[1].includes("fix-repo")){
    return {
      name: body_content[2].split(":")[1].trim(),
      fix_repo: true
    }
  }else{
    return{
      topic: body_content[1].split(":")[1].trim(),
      min_star: +body_content[2].split(":")[1],
      total_pr: +body_content[3].split(":")[1],
      fix_repo: false
    }  
  }
}
//TODO: update logic to get details using key and value pair for details

async function getRepoWithWorkflow(client:any,topic:string){  
  const repoArr=await client.rest.search.code({
  q:topic+" path:.github/workflows",
  per_page:5,
  page:CURR_PAGE,
  order: "asc",
  sort: "indexed"
  })
  CURR_MATCH %= 6
  return repoArr
}
  
async function getRepoStars(client:any, owner:string, repo:string){
  const repo_details = await client.rest.repos.get({owner:owner,repo:repo})
  return repo_details.data.stargazers_count
}

type GetRepoContentResponseDataFile = components["schemas"]["content-file"]
export async function getFile({client,owner, repo, path, branchName= "master"}:{client:any,owner:string, repo:string, path:string, branchName?:string}){
  const {data} =  await client.rest.repos.getContent({owner: owner, repo: repo,path: path, ref: `heads/${branchName}`})
  if (!Array.isArray(data)) {
    const workflow = data as GetRepoContentResponseDataFile

    if (typeof workflow.content !== undefined) {
      return Buffer.from(workflow.content, "base64").toString() // b64 decoding before returning
    }
  }else{
    core.setFailed("not a file path...")
  }   
}
  
// check whether the pr is already created or not
async function alreadyCreated(client:any, owner:string, repo:string){
  // whether pr already created or not (change to all when using in secureworkflow repo)
  const pr = await client.rest.pulls.list({owner:owner,repo:repo,state:"open"})
  return (pr.data.length > 0 ? true:false)
}
  
// get good matches
export async function getGoodMatch(client:any, topic:string, min_star:number){
  while(true){
    const repoArr = await getRepoWithWorkflow(client,topic)
    while(CURR_MATCH<6){
      let owner = repoArr.data.items[CURR_MATCH].repository.owner.login
      let repo = repoArr.data.items[CURR_MATCH].repository.name
      let path = repoArr.data.items[CURR_MATCH].path
      if(await getRepoStars(client,owner,repo)>=min_star && !(await alreadyCreated(client,owner,repo))){
        const content = await getFile({client,owner,repo,path})
        return{
          owner:owner,
          repository:repo,
          path:path,
          content:content
        } 
      }
      CURR_MATCH++
    }
    CURR_PAGE++
  }
}

// TODO: log all matches
// TODO: log reason to skip matches

type GetRepoContentResponseDataFolder = components["schemas"]["content-directory"]
export async function getFilesInFolder(client:any, owner:string, repo:string){
  const {data} =  await client.rest.repos.getContent({owner: owner, repo: repo,path: ".github/workflows"})
  const folder = data as GetRepoContentResponseDataFolder
  const worklflows = []
  let curr=0
  while(curr<folder.length){
    worklflows.push(folder[curr].name)
    curr++
  }
  return worklflows
}