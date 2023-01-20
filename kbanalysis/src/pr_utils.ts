import { mkdirSync, appendFileSync } from "fs";


export function createActionYaml(owner:string, repo:string, content:string){
    let path = `knowledge-base/actions/${owner.toLocaleLowerCase()}/${repo.toLocaleLowerCase()}`
    let repo_file = `action-security.yml`
    let full_path = `${path}/${repo_file}`
  
    mkdirSync(path, {recursive: true})
    appendFileSync(full_path, content, {flag:"a+"});
}