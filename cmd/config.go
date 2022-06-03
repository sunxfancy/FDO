package cmd

import (
	"io/ioutil"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Source string
	Args   []string
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

func StoreConfig(file string, config Config) {
	c, err := yaml.Marshal(config)
	if err != nil {
		panic(err)
	}
	err = ioutil.WriteFile(file, c, 0644)
	if err != nil {
		panic(err)
	}
}
