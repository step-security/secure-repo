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
	Actions        string `yaml:"actions" json:"actions"`
	Checks         string `yaml:"checks" json:"checks"`
	Contents       string `yaml:"contents" json:"contents"`
	Deployments    string `yaml:"deployments" json:"deployments"`
	Packages       string `yaml:"packages" json:"packages"`
	PullRequests   string `yaml:"pull-requests" json:"pull-requests"`
	Issues         string `yaml:"issues" json:"issues"`
	SecurityEvents string `yaml:"security-events" json:"security-events"`
	Statuses       string `yaml:"statuses" json:"statuses"`
}

type Action struct {
	Name         string      `yaml:"name" json:"name"`
	DefaultToken string      `yaml:"default-token" json:"default-token"`
	EnvKey       string      `yaml:"env-key" json:"env-key"`
	Permissions  Permissions `yaml:"permissions" json:"permissions"`
}

type ActionPermissions struct {
	Actions Actions `yaml:"actions"`
}

type Actions map[string]Action

type GitHubContent struct {
	Content string `json:"content"`
}
