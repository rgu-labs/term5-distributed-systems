package main

type Config struct {
	Env        string `default:"dev"`
	ServerType string `default:"tcp"`
	Port       int    `default:"8080"`
}

func (c *Config) GetEnv() string { return c.Env }
