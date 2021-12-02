package main

import (
	"errors"
	"strings"
)

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

type ActionMetadata struct {
	Name        string      `yaml:"name"`
	GitHubToken GitHubToken `yaml:"github-token"`
}

type ActionInput struct {
	Input     string `yaml:"input"`
	IsDefault bool   `yaml:"is-default"`
}

type GitHubToken struct {
	ActionInput             ActionInput            `yaml:"action-input"`
	EnvironmentVariableName string                 `yaml:"environment-variable-name"`
	Permissions             ActionScopePermissions `yaml:"permissions"`
}

type ActionScopePermissions struct {
	Scopes map[string]ActionScopePermission
}

type ActionScopePermission struct {
	Permission string
	Reason     string
	Expression string
}

type ActionPermissions struct {
	Actions Actions `yaml:"actions"`
}

type Actions map[string]Action

type GitHubContent struct {
	Content string `json:"content"`
}

func (p *ActionScopePermissions) UnmarshalYAML(unmarshal func(interface{}) error) error {
	mstr := make(map[string]string)
	err := unmarshal(&mstr)
	if err != nil {
		return ErrInvalidValue
	}

	scopeMap := make(map[string]string)
	reasonMap := make(map[string]string)
	ExpressionMap := make(map[string]string)
	actionScopePermissionMap := make(map[string]ActionScopePermission)

	for k, v := range mstr {
		if strings.HasSuffix(k, "-reason") {
			scope := strings.Split(k, "-reason")[0]
			reasonMap[scope] = v
		} else if strings.HasSuffix(k, "-if") {
			scope := strings.Split(k, "-if")[0]
			ExpressionMap[scope] = v
		} else {
			scopeMap[k] = v
		}
	}

	for k, v := range scopeMap {
		reason := reasonMap[k]
		expression := ExpressionMap[k]
		actionScopePermissionMap[k] = ActionScopePermission{Permission: v, Reason: reason, Expression: expression}
	}

	p.Scopes = actionScopePermissionMap
	return nil

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
