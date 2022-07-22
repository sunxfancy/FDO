package cmd

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Source  string
	Args    []string
	lto     string
	Profile string
	TestCfg string
	Install bool
	ipra    bool
	DryRun  bool
}

func LoadConfig(file string) Config {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var config Config
	yaml.Unmarshal(c, &config)
	return config
}

func (config Config) StoreConfig(file string) {
	c, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(file, c, 0644)
	if err != nil {
		panic(err)
	}
}

type TestScript struct {
	Commands      []string
	Binary        string
	ClangPath     string
	PropellerPath string
	RegPath       string
}

func LoadTestScript(file string) TestScript {
	c, err := ioutil.ReadFile(file)
	if err != nil {
		panic(err)
	}
	var script TestScript
	yaml.Unmarshal(c, &script)
	return script
}

func StoreTestScript(file string, script TestScript) {
	c, err := yaml.Marshal(script)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(file, c, 0644)
	if err != nil {
		panic(err)
	}
}
