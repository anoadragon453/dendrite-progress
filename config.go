package main

import (
	"github.com/BurntSushi/toml"
	"os"
	"io/ioutil"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

// Config is a struct holding all global configuration values
type Config struct {
	Git     Git
	Webhook Webhook
	DB      Database `toml:"database"`
	HTTP    HTTP

	DendriteTestfileURL string `toml:"dendrite_testfile_url"`
	LogLevel            log.Level `toml:"log_level"`
}

// HTTP is a configuration struct containing information about http
type HTTP struct {
	Port int
}

// Git is a configuration struct containing information about git
// repositories
type Git struct {
	SytestURL       string `toml:"sytest_url"`
	SytestDirectory string `toml:"sytest_directory"`
}

// Webhook is a configuration struct containing information about webhooks
type Webhook struct {
	DendriteSecret string `toml:"dendrite_secret"`
	SytestSecret   string `toml:"sytest_secret"`
}

// Database is a configuration struct containing information about the database
type Database struct {
	Path string
}

// loadConfig is a function that loads the contents of the config file, parses
// and processes it, and returns a filled Config type, and any encountered
// errors
func loadConfig() (Config, error) {
	var config Config

	// Read config file
	configFile, err := os.Open("config.toml")
	if err != nil {
		return config, err
	}
	configFileContents, err := ioutil.ReadAll(configFile)
	if err != nil {
		return config, err
	}

	// Decode file contents as toml
	_, err = toml.Decode(string(configFileContents), &config)
	if err != nil {
		return config, err
	}
	return processConfig(config)
}

// processConfig is a function that does any post-processing on the config file
// values. For example, turning relative paths absolute.
func processConfig(config Config) (Config, error) {
	var err error

	// Absolute relative paths
	config.Git.SytestDirectory, err = filepath.Abs(config.Git.SytestDirectory)
	if err != nil {
		return config, err
	}
	config.DB.Path, err = filepath.Abs(config.DB.Path)
	return config, err
}
