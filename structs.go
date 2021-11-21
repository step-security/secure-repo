package main

type Workflow struct {
	Name        string      `yaml:"name"`
	Permissions Permissions `yaml:"permissions"`
	//On   string `yaml:"on"`
	Jobs Jobs `yaml:"jobs"`
}
type Step struct {
	Run  string `yaml:"run"`
	Uses string `yaml:"uses"`
	With With   `yaml:"with"`
	Env  Env    `yaml:"env"`
}
type Job struct {
	Permissions Permissions `yaml:"permissions"`
	// RunsOn      []string    `yaml:"runs-on"`
	Steps []Step `yaml:"steps"`
}

type Jobs map[string]Job
type With map[string]string
type Env map[string]string

// For action-permissions.yml

type Permissions struct {
	Actions        string `yaml:"actions"`
	Checks         string `yaml:"checks"`
	Contents       string `yaml:"contents"`
	Deployments    string `yaml:"deployments"`
	Packages       string `yaml:"packages"`
	PullRequests   string `yaml:"pull-requests"`
	Issues         string `yaml:"issues"`
	SecurityEvents string `yaml:"security-events"`
	Statuses       string `yaml:"statuses"`
}

type Action struct {
	Name         string      `yaml:"name"`
	DefaultToken string      `yaml:"default-token"`
	EnvKey       string      `yaml:"env-key"`
	Permissions  Permissions `yaml:"permissions"`
}

type ActionPermissions struct {
	Actions Actions `yaml:"actions"`
}

type Actions map[string]Action

type GitHubContent struct {
	Content string `json:"content"`
}
