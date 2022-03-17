import {info} from "@actions/core"
import { searchEndpoints } from "./endpoints"

export function isKBIssue(title:String){
    const prefix = "[KB] Add KB for" // pattern to check, for KB issue
    const index = title.indexOf(prefix) 
    return index === 0 // for valid KB issue; index of prefix is always 0

}

export function getAction(title:String){
    const splits = title.split(" ")
    const name = splits.pop()
    return name !== undefined ? name !== "" ? name : "not_present" : "not_present"
}

export function validateAction(client, action:String){
    // Function that will verify the existence of action
}

async function getFile(client:any, owner:String, repo:String, path:String){

    const action_data =  await client.rest.repos.getContent({owner: owner, repo: repo,path: path})
    const encoded_content = action_data.data["content"].split("\n")
    const content = encoded_content.join("")
    return Buffer.from(content, "base64").toString() // b64 decoding before returning

}

export async function readFile(client, owner, repo, path){

    const norm = normalizeRepo(repo)
    const content = await getFile(client, owner, norm.repo, norm.path+"/"+path)
    return content

}

export function normalizeRepo(repo:String){
    let true_repo:String = ""
    let path:String = ""
    if(repo.indexOf("/") > 0){
        // nested repo.
        const repo_split = repo.split("/")
        true_repo = repo_split[0] // first part is true_name of repo
        path = repo_split.slice(1,).join("/")
    }else{
        true_repo = repo
    }

    return {repo:true_repo, path:path}
}

export async function checkDependencies(client,owner:String, repo:String){
    // Function for analyzing package.json
    // Note: Use this function only if the action is Node based.
    const package_content = await readFile(client, owner, repo, "package.json")
    const deps = ["github", "octokit"] // if any one of these is present, check passes
    for(let dep of deps){
        if(package_content.indexOf(dep) !== -1){
            return true
        }
    }
    return false
}

export async function getActionYaml(client: any, owner: String, repo: String){
    
    const norm = normalizeRepo(repo)
    const action_data =  await getFile(client,owner, norm.repo,norm.path+"/action.yml")
    return action_data

}

export async function getReadme(client:any, owner:String, repo:String){
    const norm = normalizeRepo(repo)
    let readme = ""
    try{
        readme =  await getFile(client,owner, norm.repo,norm.path+"/README.md")
    }catch{
        readme = null
    }
    return readme
}

export function getRunsON(content: String){
    const usingIndex = content.indexOf("using:")
    const usingString = content.substring(usingIndex+6, usingIndex+6+10)
    return usingString.indexOf("node") > -1 ? "Node" : usingString.indexOf("docker") > -1 ? "Docker" : "Composite"
}

export async function findToken(content:String){
    // if token is not found, returns a list; otherwise return null
    // TODO: always handle null; when used this function.
    const pattern = /((github|repo|gh|pat)[_,-](token|tok)|(token|oidc))/gmi
    const matches = content.match(pattern)
    return matches !== null ? matches.filter((value, index, self)=> self.indexOf(value)===index) : null // returning only unique matches.
}

export function printArray(arr, header){
    info(`${header}`)
    for(let elem of arr){
        info(`-->${elem}`)
    }
}

export async function findEndpoints(client, owner:String, repo:String, src_files:String[]){

    let perms = {}
    for(let src of src_files){
        let cont = await readFile(client, owner, repo, src)
        let deps = await searchEndpoints(cont)
        if(deps !== {}){
            let keys = Object.keys(deps)
            for(let k of keys){
                perms[k] = deps[k]
            }
        }
    }
    return perms
}


export function permsToString(perms:Object){
    const keys = Object.keys(perms)
    let out = ""
    let header = "|Endpoint | Permission|\n"
    header    += "|---------| ----------|\n" 
    out += header
    for(let k of keys){
        out += `${k} | ${perms[k]}\n`
    }
    return out
}

export function isValidLang(lang:String){
    // issue#10
    const valid_string =  "javascripttypescript"
    return valid_string.indexOf(lang.toLocaleLowerCase()) !== -1
}

export async function comment(client, repos, issue_id, body){
    await client.rest.issues.createComment({
        ...repos,
        issue_number: Number(issue_id),
        body: body
    })
}

export function getTokenInput(action_yml:String, tokens_found:String[]){

    let output = []
    for(let tok of tokens_found){
        if(action_yml.indexOf(tok+":") !== -1){
            output.push(tok)
        }
    }
    
    return output.length !== 0 ? output[0] : "env_var"

}

export function actionSecurity(data:{name:string, token_input:string, perms:{}}){

    let template = ["```yaml"]
    template.push(`${data.name}`)
    template.push("github-token:")
    template.push(`  ${data.token_input}`)
    template.push("  permissions:")
    for(let perm_key of Object.keys(data.perms)){
        template.push(`    ${perm_key}: ${data.perms[perm_key]}`)
    }

    template.push("```\n")

    return template.join("\n")


}

export function normalizePerms(perms:{}){

    // const mapping = {"pul"}
    const mapping = {
        'actions':"actions", 
        'checks':"checks",
        'git':"contents",
        'issues':"issues",
        'meta':"metadata",            
        'pulls':"pull-requests",
        'repos':"contents",
    }

    let norm_perms = {}
    for(let k of Object.keys(perms)){
        const prefix = mapping[k.split(".")[0]]
        if(norm_perms[prefix] !== undefined){
            // key already exists
            if(norm_perms[prefix] !== "write"){
                norm_perms[prefix] = perms[k]
            }
        }else{
            norm_perms[prefix] = perms[k]
        }
        
    }

    return norm_perms;

}