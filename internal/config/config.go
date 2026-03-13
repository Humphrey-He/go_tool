package config

type Config struct {
	Schema SchemaConfig `toml:"schema" json:"schema"`
	Scan   ScanConfig   `toml:"scan" json:"scan"`
	Rules  RulesConfig  `toml:"rules" json:"rules"`
	Output OutputConfig `toml:"output" json:"output"`
}

type SchemaConfig struct {
	Driver string `toml:"driver" json:"driver"`
	DSN    string `toml:"dsn" json:"dsn"`
	DDL    string `toml:"ddl" json:"ddl"`
}

type ScanConfig struct {
	Workspace string   `toml:"workspace" json:"workspace"`
	Include   []string `toml:"include" json:"include"`
	Exclude   []string `toml:"exclude" json:"exclude"`
	Workers   int      `toml:"workers" json:"workers"`
}

type RulesConfig struct {
	Enable  []string `toml:"enable" json:"enable"`
	Disable []string `toml:"disable" json:"disable"`
}

type OutputConfig struct {
	JSON bool `toml:"json" json:"json"`
}

func DefaultConfig() Config {
	return Config{
		Schema: SchemaConfig{
			Driver: "mysql",
		},
		Scan: ScanConfig{
			Include: []string{"**/*.go"},
			Exclude: []string{"**/vendor/**", "**/.git/**"},
			Workers: 0,
		},
		Output: OutputConfig{
			JSON: true,
		},
	}
}
