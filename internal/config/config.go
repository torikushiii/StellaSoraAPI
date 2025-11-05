package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

const (
	defaultServerAddr = ":8080"
	defaultMongoURI   = "mongodb://localhost:27017"
	defaultMongoDB    = "stella-sora"
)

type Config struct {
	Server ServerConfig `yaml:"server"`
	Mongo  MongoConfig  `yaml:"mongo"`
}

type ServerConfig struct {
	Addr string `yaml:"addr"`
}

type MongoConfig struct {
	URI      string `yaml:"uri"`
	Database string `yaml:"database"`
}

func Load(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Server.Addr == "" {
		cfg.Server.Addr = defaultServerAddr
	}
	if cfg.Mongo.URI == "" {
		cfg.Mongo.URI = defaultMongoURI
	}
	if cfg.Mongo.Database == "" {
		cfg.Mongo.Database = defaultMongoDB
	}

	return cfg, nil
}
