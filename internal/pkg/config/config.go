package config

import (
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type Config struct {
	DbConnectString *string `yaml:"DbConnectString"`
}

func LoadConfig(configStr []byte) Config {
	var config Config
	err := yaml.Unmarshal(configStr, &config)
	if err != nil {
		logrus.Fatalf("failed to parse configuration file %v: %v", string(configStr), err)
	}
	return config
}

type OrderBy struct {
	Column string `yaml:"column"`
	Order  string `yaml:"order"`
}

type TablePopulateDef struct {
	Type    string    `yaml:"type"`
	Name    string    `yaml:"name"`
	By      []string  `yaml:"by"`
	Orderby []OrderBy `yaml:"orderby"`
}

type TableDef struct {
	Owner    string             `yaml:"owner"`
	Name     string             `yaml:"name"`
	Populate []TablePopulateDef `yaml:"populate"`
}

type ParameterDef struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

type QueryDef struct {
	Name       string         `yaml:"name"`
	Parameters []ParameterDef `yaml:"parameters"`
	Query      string         `yaml:"query"`
}

type ModelConfig struct {
	Tables  []TableDef `yaml:"tables"`
	Queries []QueryDef `yaml:"queries"`
}

func LoadModelConfig(modelDefStr []byte) ModelConfig {
	var modelConfig ModelConfig
	err := yaml.Unmarshal(modelDefStr, &modelConfig)
	if err != nil {
		logrus.Fatalf("failed to parse model %v: %v", string(modelDefStr), err)
	}
	logrus.Debugf("Parse model: %+v", modelConfig)
	return modelConfig
}
