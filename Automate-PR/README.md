# Automate PR
This action finds repos containing workflows without permissions and the uses StepSecurity's open source tool secure-workflow to secure the workflow and then creates PR to those repos with added permissions.

>motivation is to avoid supply chain attacks

## Usage
>Note : This action requires, `contents:read`, `actions:write` & `issues:write` permissions, so make sure to declare them in `job_permissions`.

Just add below snippet in your `workflow's job steps` to put this action into work.

```yml
    steps:
      - name: Automate PR
        uses: Devils-Knight/Automate-PR@master
        with:
          github-token: ${{secrets.GITHUB_TOKEN }}
          issue-id: ${{ github.event.issue.number}}

```
Add Below snipet to the job to trigger the workflow whenever an issue is created with `Automate` label.
```yml
if: github.event.label.name == 'Automate'
```

Now, whenever a issue with `Automate` label is created, this action will perform `Automation` and will create `Pull-request` to workflow repo with added permissions.

---

### Add below snipet to provide details for the workflow :
```yml
topic : ${topic relevant}
min_star : ${minimum number of stars}
total_pr : ${total number of workflows to be secured}
```
### If you want to secure all the workflows for a specified repo add below snipet :
```yml
fix-repo
name : ${owner}/${repo}
```
>Above snipet will secure all the workflows for the repo but if some workflows failed due to missing KB's you can `restart` the workflow after removing and readding `Automate` label to the issue after adding those missing KB's