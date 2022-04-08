import * as core from "@actions/core"
import axios from 'axios';

interface SecureWorkflowReponse {
	FinalOutput: string,
	IsChanged : boolean,
	HasErrors : boolean,
	AlreadyHasPermissions : boolean,
	IncorrectYaml : boolean,
	JobErrors : JobError[],
	MissingActions : string[]
}

interface JobError {
	JobName : string,
	Errors : string[]
}


export async function getResponse (payload : any){
  const apiClient = axios.create({
    baseURL: 'https://sa0mwebuda.execute-api.us-west-2.amazonaws.com',
    responseType: 'json',
    headers: {
      'Content-Type': 'text/plain'
    }
  });

  const response = await apiClient.post<SecureWorkflowReponse>('/v1/secure-workflow?addHardenRunner=false&pinActions=false&', payload);
  const user = response.data;
  return user;
};
