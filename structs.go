package main

import "errors"

var (
	ErrInvalidValue = errors.New("invalid value for field 'permissions'")
)

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
	Scopes   map[string]string
	ReadAll  bool
	WriteAll bool
	IsSet    bool
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

func (p *Permissions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	mstr := make(map[string]string)
	if err := unmarshal(&mstr); err == nil {
		p.Scopes = mstr
		p.IsSet = true
		return nil
	}

	permString := ""
	if err := unmarshal(&permString); err == nil {
		if permString == "read-all" {
			p.ReadAll = true
		} else if permString == "write-all" {
			p.WriteAll = true
		}
		p.IsSet = true
		return nil
	}

	return ErrInvalidValue
}
