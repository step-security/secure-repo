# KBAnalysis
An action that performs `automatic_analysis` of `action` present in `issue_title`, if it finds that the `issue` is a valid `KB issue`, whenever a new issue is opened.

>The motive behind the  development of this action is to reduce the `analysis time`, while figuring `token permissions`.

## Usage
>Note : This action requires, `contents:read` & `issues:write` permissions, so make sure to declare them in `job_permissions`.

Just add below snippet in your `workflow's job steps` to put this action into work.

```yml
    steps:
      - name: KBAnalysis
        uses: h0x0er/kbanalysis@master
        with:
          github-token: ${{secrets.GITHUB_TOKEN }}
          issue-id: ${{ github.event.issue.number}}

```
Now, whenever a `valid KB issue is created`, this action will perform `analysis` and will create a `comment` in that issue.

## Working

