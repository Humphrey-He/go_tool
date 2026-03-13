package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"runtime"

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

	reg := rules.NewRegistry()
	reg.Register(&rules.FieldNotExistsRule{})

	collector := &analyzer.ASTInspector{}
	parserImpl := &parser.VitessParser{}
	loader := &schema.DDLLoader{Path: cfg.Schema.DDL}

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
