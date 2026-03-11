package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Routes []Route      `yaml:"routes"`
}

type ServerConfig struct {
	Address string `yaml:"address"`
}

type Route struct {
	Path          string          `yaml:"path"`
	Method        string          `yaml:"method"`
	Async         bool            `yaml:"async"`
	Response      *StaticResponse `yaml:"response"`
	Builtin       *BuiltinAction  `yaml:"builtin"`
	AsyncResponse *StaticResponse `yaml:"async_response"`
	Exec          *ExecAction     `yaml:"exec"`
}

type StaticResponse struct {
	Status  int               `yaml:"status"`
	Headers map[string]string `yaml:"headers"`
	Body    any               `yaml:"body"`
}

type ExecAction struct {
	Command        string            `yaml:"command"`
	Args           []string          `yaml:"args"`
	Dir            string            `yaml:"dir"`
	Env            map[string]string `yaml:"env"`
	TimeoutSeconds int               `yaml:"timeout_seconds"`
	PassBody       bool              `yaml:"pass_body"`
}

type BuiltinAction struct {
	Name string `yaml:"name"`
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	if len(cfg.Routes) == 0 {
		return nil, errors.New("config must contain at least one route")
	}

	return &cfg, validate(&cfg)
}

func validate(cfg *Config) error {
	for i, route := range cfg.Routes {
		if route.Path == "" {
			return fmt.Errorf("routes[%d]: path is required", i)
		}
		actions := 0
		if route.Response != nil {
			actions++
		}
		if route.Builtin != nil {
			actions++
		}
		if route.Exec != nil {
			actions++
		}
		if actions == 0 {
			return fmt.Errorf("routes[%d]: one of response, builtin, or exec must be set", i)
		}
		if actions > 1 {
			return fmt.Errorf("routes[%d]: response, builtin, and exec are mutually exclusive", i)
		}
		if route.Builtin != nil && route.Builtin.Name == "" {
			return fmt.Errorf("routes[%d]: builtin.name is required", i)
		}
		if route.Exec != nil && route.Exec.Command == "" {
			return fmt.Errorf("routes[%d]: exec.command is required", i)
		}
		if route.Async && route.Exec == nil && route.Builtin == nil {
			return fmt.Errorf("routes[%d]: async requires builtin or exec", i)
		}
		if route.AsyncResponse != nil && !route.Async {
			return fmt.Errorf("routes[%d]: async_response requires async=true", i)
		}
	}

	return nil
}
