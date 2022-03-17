import {exec } from "child_process";
import * as core from "@actions/core";
import { writeFile } from "fs";



function terminal(cmd:string){
    exec(cmd, async (error, stdout, stderr)=>{

        if(error){core.warning(`Error occurred: ${error}`)}
        if(stderr){core.warning(`Error occurred: ${stderr}`)}
        if(stdout){core.info(`Output: ${stdout}`)}


    })  
}

export async function createPR(content:string, path:String){
    path = path.toLocaleLowerCase();
    terminal(`mkdir -p ${path}`)
    terminal(`touch ${path}/action-security.yml`)
    terminal(`ls ${path}`)
    writeFile(`${path}/action-security.yml`, content, (err)=>{
        if(err){
            core.warning("error occurred while creating action-security.yml")
        }
    })

}