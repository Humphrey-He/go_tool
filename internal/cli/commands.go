package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"
	"time"

	"go_tool/internal/config"
	"go_tool/internal/core/analyzer"
	"go_tool/internal/output"
	"go_tool/internal/parser"
	"go_tool/internal/report"
	"go_tool/internal/rules"
	"go_tool/internal/schema"
)

const defaultConfigName = ".sqlsafelint.json"

func Run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		printUsage()
		return nil
	}

	switch args[0] {
	case "init":
		return runInit(args[1:])
	case "scan":
		return runScan(ctx, args[1:])
	case "help", "-h", "--help":
		printUsage()
		return nil
	default:
		printUsage()
		return fmt.Errorf("unknown command: %s", args[0])
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigName, "config file path")
	if err := fs.Parse(args); err != nil {
		return err
	}

	if err := config.WriteDefault(*configPath); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(os.Stdout, "Initialized config at %s\n", *configPath)
	return nil
}

func runScan(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("scan", flag.ContinueOnError)
	configPath := fs.String("config", defaultConfigName, "config file path")
	jsonOnly := fs.Bool("json", false, "output JSON only")
	if err := fs.Parse(args); err != nil {
		return err
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return err
	}

	if cfg.Scan.Workers <= 0 {
		cfg.Scan.Workers = runtime.GOMAXPROCS(0)
	}
	if cfg.Scan.TimeoutMs <= 0 {
		cfg.Scan.TimeoutMs = 500
	}

	ctx, cancel := context.WithTimeout(ctx, time.Duration(cfg.Scan.TimeoutMs)*time.Millisecond)
	defer cancel()

	reg := rules.NewRegistry()
	reg.Register(&rules.FieldNotExistsRule{})
	reg.Register(&rules.MissingIndexRule{SearchPath: cfg.Schema.SearchPath})

	collector := &analyzer.ASTInspector{}
	parserImpl := &parser.VitessParser{}
	loader := resolveSchema(cfg)

	eng := analyzer.Engine{
		Collector: collector,
		Parser:    parserImpl,
		Schema:    loader,
		Rules:     reg,
	}

	diags, err := eng.Analyze(ctx, cfg)
	if err != nil {
		return err
	}

	if err := output.WriteJSON(os.Stdout, diags); err != nil {
		return err
	}

	if !*jsonOnly {
		output.WriteSummary(os.Stdout, diags)
	}

	if hasErrors(diags) {
		return errors.New("scan failed")
	}

	return nil
}

func resolveSchema(cfg config.Config) schema.Loader {
	cache := schema.Cache{Path: cfg.Schema.CachePath, TTL: time.Duration(cfg.Schema.CacheTTL) * time.Second}
	switch cfg.Schema.Driver {
	case "postgres":
		return schema.CachedLoader{Loader: &schema.PostgresLoader{DSN: cfg.Schema.DSN, SearchPath: cfg.Schema.SearchPath}, Cache: cache, Source: "postgres", SearchPath: cfg.Schema.SearchPath}
	case "ddl":
		return schema.CachedLoader{Loader: &schema.DDLLoader{Path: cfg.Schema.DDL}, Cache: cache, Source: "ddl", SearchPath: cfg.Schema.SearchPath}
	default:
		return schema.CachedLoader{Loader: &schema.DDLLoader{Path: cfg.Schema.DDL}, Cache: cache, Source: "ddl", SearchPath: cfg.Schema.SearchPath}
	}
}

func hasErrors(diags []report.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == report.SeverityError {
			return true
		}
	}
	return false
}

func printUsage() {
	_, _ = fmt.Fprintln(os.Stdout, "sqlsafelint <command> [flags]")
	_, _ = fmt.Fprintln(os.Stdout, "commands: init, scan")
}
