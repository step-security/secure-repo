import * as core from "@actions/core";
import * as github from "@actions/github";
import { existsSync} from "fs";
import { exit } from "process";
import { createActionYaml } from "./pr_utils";
import {
  getActionYaml,
  findToken,
  printArray,
  getRunsON,
  getReadme,
  checkDependencies,
  findEndpoints,
  permsToString,
  isValidLang,
  actionSecurity,
  getTokenInput,
  normalizePerms,
} from "./utils";

try {
  const token = core.getInput("github-token");
  const client = github.getOctokit(token); // authenticated octokit

  const repos = github.context.repo;
  const event = github.context.eventName;

  if (event === "workflow_dispatch") {
    let owner = core.getInput("owner");
    let repo = core.getInput("repo");

    if (
      existsSync(
        `knowledge-base/actions/${owner.toLocaleLowerCase()}/${repo.toLocaleLowerCase()}`
      )
    ) {
      core.setFailed(`[!] KB already exists for action ${owner}/${repo}`);
      exit(1);
    }

    core.info("[+] Need to perform analysis");
    let issue_id = -1; // PR_ ID
    let marker = `${owner}/${repo}`;
    try {
      let repos_result = await client.rest.pulls.list({
        owner: repos.owner,
        repo: repos.repo,
        state: "open",
        per_page: 100,
        base: "knowledge-base",
      });
      for (let pull of repos_result.data) {
        if (pull.title.indexOf(marker) > -1) {
          issue_id = pull.number;
          break;
        }
      }
    } catch (err) {
      core.setFailed(err);
    }
    if (issue_id > 0) {
      core.setFailed(
        `[+] PR-${issue_id} already exists for the action ${owner}/${repo}`
      );
      exit(1);
    }

    const target_owner = owner;
    const target_repo = repo;
    const action_name = `${owner}/${repo}`;

    if (
      existsSync(
        `knowledge-base/actions/${target_owner.toLocaleLowerCase()}/${target_repo.toLocaleLowerCase()}/action-security.yml`
      )
    ) {
      core.setFailed("[+] Not performing analysis as issue is already analyzed");
      exit(1);
    }

    core.info("===== Performing analysis:  =====");

    let repo_info;
    try {
      repo_info = await client.rest.repos.get({
        owner: target_owner,
        repo: target_repo.split("/")[0],
      }); // info related to repo.
    } catch (err) {
      core.setFailed(`[+] Failed to fetch repo info: ${err}`);
      exit(1);
    }

    let lang: String = "";
    try {
      const langs = await client.rest.repos.listLanguages({
        owner: target_owner,
        repo: target_repo.split("/")[0],
      });
      lang = Object.keys(langs.data)[0]; // top language used in repo
    } catch (err) {
      lang = "NOT_FOUND";
    }

    core.info(`Action: ${action_name}`);
    core.info(`Top language: ${lang}`);
    core.info(`Stars: ${repo_info.data.stargazers_count}`);
    core.info(`Private: ${repo_info.data.private}`);

    try {
      const action_data = await getActionYaml(
        client,
        target_owner,
        target_repo
      );
      const readme_data = await getReadme(client, target_owner, target_repo);

      const start = action_data.indexOf("name:");
      const action_yaml_name = action_data.substring(
        start,
        start + action_data.substring(start).indexOf("\n")
      );

      const action_type = getRunsON(action_data);
      core.info(`Action Type: ${action_type}`);

      // determining if token is being set by default
      const pattern = /\${{.*github\.token.*}}/; // default github_token pattern
      const is_default_token = action_data.match(pattern) !== null;

      let matches: String[] = []; // // list holding all matches.
      const action_matches = await findToken(action_data);
      if (readme_data !== null) {
        const readme_matches = await findToken(readme_data);
        if (readme_matches !== null) {
          matches.push(...readme_matches); // pushing readme_matches in main matches.
        }
      }
      if (action_matches !== null) {
        matches.push(...action_matches);
      }
      if (matches.length === 0) {
        // no github_token pattern found in action_file & readme file
        core.warning("Action doesn't contains reference to github_token");
        const template = `\n\`\`\`yaml\n${action_yaml_name} # ${
          target_owner + "/" + target_repo
        }\n# GITHUB_TOKEN not used\n\`\`\`\n`;
        const action_yaml_content = `${action_yaml_name} # ${
          target_owner + "/" + target_repo
        }\n# GITHUB_TOKEN not used\n`;
        await createActionYaml(target_owner, target_repo, action_yaml_content);
      } else {
        // we found some matches for github_token
        matches = matches.filter(
          (value, index, self) => self.indexOf(value) === index
        ); // unique matches only.
        core.info("Pattern Matches: " + matches.join(","));

        if (
          lang === "NOT_FOUND" ||
          action_type === "Docker" ||
          action_type === "Composite"
        ) {
          // Action is docker or composite based no need to perform token_queries
          await createActionYaml(
            owner,
            repo,
            "# Action is docker or composite based.\n#Need to perform manual analysis"
          );
        } else {
          let action_security_yaml = ""; // content of action-yaml file

          // Action is Node Based
          let is_used_github_api = false;
          if (isValidLang(lang)) {
            is_used_github_api = await checkDependencies(
              client,
              target_owner,
              target_repo
            );
          }
          core.info(`Github API used: ${is_used_github_api}`);
          let paths_found = []; // contains url to files
          let src_files = []; // contains file_paths relative to repo.

          for (let match of matches) {
            const query = `${match}+in:file+repo:${target_owner}/${target_repo}+language:${lang}`;
            const res = await client.rest.search.code({ q: query });

            const items = res.data.items.map((item) => item.html_url);
            const src = res.data.items.map((item) => item.path);

            paths_found.push(...items);
            src_files.push(...src);
          }

          const filtered_paths = paths_found.filter(
            (value, index, self) => self.indexOf(value) === index
          );
          src_files = src_files.filter(
            (value, index, self) => self.indexOf(value) === index
          ); // filtering src files.
          core.info(`Src File found: ${src_files}`);

          const valid_input = getTokenInput(action_data, matches);
          let token_input =
            valid_input !== "env_var"
              ? `action-input:\n    input: ${valid_input}\n    is-default: ${is_default_token}`
              : `environment-variable-name: <FigureOutYourself>`;

          if (is_used_github_api) {
            if (src_files.length !== 0) {
              const perms = await findEndpoints(
                client,
                target_owner,
                target_repo,
                src_files
              );
              if (perms !== {}) {
                let str_perms = permsToString(perms);
                core.info(`${str_perms}`);
                action_security_yaml += actionSecurity({
                  name: action_yaml_name,
                  token_input: token_input,
                  perms: normalizePerms(perms),
                });
              }
            }
          }
          printArray(filtered_paths, "Paths Found: ");
          try {
            if(action_security_yaml.length === 0){
                action_security_yaml += "# Error in determining permissions"
            }
            await createActionYaml(owner, repo, action_security_yaml);
          } catch (err) {
            core.info(`Unable to write action-security.yaml: ${err}`);
          }
        }
      }
    } catch (err) {
      core.setFailed(err);
      exit(1);
    }

    exit(0);
  }
} catch (err) {
  core.setFailed(err);
}
