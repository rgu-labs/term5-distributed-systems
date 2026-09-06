package main

type Config struct {
	Env  string `default:"dev"`
	Port int    `default:"8080"`
}

func (c *Config) GetEnv() string { return c.Env }
