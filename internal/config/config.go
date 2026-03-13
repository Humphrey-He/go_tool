package config

type Config struct {
	Schema SchemaConfig `toml:"schema" json:"schema" yaml:"schema"`
	Scan   ScanConfig   `toml:"scan" json:"scan" yaml:"scan"`
	Rules  RulesConfig  `toml:"rules" json:"rules" yaml:"rules"`
	Output OutputConfig `toml:"output" json:"output" yaml:"output"`
}

type SchemaConfig struct {
	Driver     string   `toml:"driver" json:"driver" yaml:"driver"`
	DSN        string   `toml:"dsn" json:"dsn" yaml:"dsn"`
	DDL        string   `toml:"ddl" json:"ddl" yaml:"ddl"`
	SearchPath []string `toml:"search_path" json:"search_path" yaml:"search_path"`
	CacheTTL   int      `toml:"cache_ttl" json:"cache_ttl" yaml:"cache_ttl"`
	CachePath  string   `toml:"cache_path" json:"cache_path" yaml:"cache_path"`
}

type ScanConfig struct {
	Workspace string   `toml:"workspace" json:"workspace" yaml:"workspace"`
	Include   []string `toml:"include" json:"include" yaml:"include"`
	Exclude   []string `toml:"exclude" json:"exclude" yaml:"exclude"`
	Workers   int      `toml:"workers" json:"workers" yaml:"workers"`
	TimeoutMs int      `toml:"timeout_ms" json:"timeout_ms" yaml:"timeout_ms"`
}

type RulesConfig struct {
	Enable   []string `toml:"enable" json:"enable" yaml:"enable"`
	Disable  []string `toml:"disable" json:"disable" yaml:"disable"`
	Plugins  []string `toml:"plugins" json:"plugins" yaml:"plugins"`
}

type OutputConfig struct {
	JSON  bool `toml:"json" json:"json" yaml:"json"`
	Sarif bool `toml:"sarif" json:"sarif" yaml:"sarif"`
}

func DefaultConfig() Config {
	return Config{
		Schema: SchemaConfig{
			Driver:     "ddl",
			SearchPath: []string{"public"},
			CacheTTL:   300,
			CachePath:  ".sqlsafelint.schema.json",
		},
		Scan: ScanConfig{
			Include:   []string{"**/*.go"},
			Exclude:   []string{"**/vendor/**", "**/.git/**"},
			Workers:   0,
			TimeoutMs: 500,
		},
		Rules: RulesConfig{
			Enable:  []string{},
			Plugins: []string{},
		},
		Output: OutputConfig{
			JSON: true,
		},
	}
}
