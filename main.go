// package main
//
// This tool analyzes a Go project and determines which packages are affected by recent changes.
// It is designed to be used in CI pipelines or development workflows where efficient testing or linting
// of only the impacted packages is desired.
//
// Features:
//
// - Detects all .go files that have changed compared to a base branch or commit (default: origin/main).
// - Maps changed files to their corresponding Go packages.
// - Propagates affected status to packages that import the changed packages (recursively).
// - Detects if "go.mod" has changed and, if so:
//   - Parses the previous and current versions of go.mod.
//   - Compares dependencies (modules) and identifies which ones were added, removed, or had version changes.
//   - Identifies project packages that import the changed modules (transitively).
//
// - Outputs the list of all affected packages in various formats:
//   - Plain text (one package per line).
//   - JSON array of affected packages.
//   - JSON plan format that groups affected packages by application (if applicable) and lists others separately.
//
// Usage:
//
//	go run tools/dev/go-ripple/main.go [-b <base>] [-o <output>]
//
// Example:
//
//	go run tools/dev/go-ripple/main.go -b origin/main -o json
//
// Dependencies:
//
// - Git must be installed and accessible via the system PATH.
// - The project must have a valid go.mod and go.sum at its root.
// - The base reference (e.g. origin/main) must be fetchable by Git.
//
// Argument Flags:
//
// -b, --base   The Git base branch or commit to compare against. Defaults to "origin/main".
//
// This script is intended for monorepos or large Go projects where full builds or tests
// are expensive and should be scoped to only affected components.
package main

import (
	"context"
	"log"

	"github.com/tangelo-labs/go-ripple/internal/rippler"
	"github.com/tangelo-labs/go-ripple/internal/rippler/printers"
	"github.com/alexflint/go-arg"
)

// Arguments holds the command line arguments for the tool.
type Arguments struct {
	Path         string `arg:"positional" placeholder:"PATH" help:"The path to the Go project directory (holding a go.mod file). Defaults to the current directory if not specified." default:"."`
	Base         string `arg:"-b,--base" help:"The base commit or branch to compare against. This is passed to 'git diff'. Defaults to 'origin/main' if not specified." default:"origin/main"`
	OutputFormat string `arg:"-o,--output" help:"How to present the results, valid options are: plain, json, test-plan, test-matrix, explain" default:"plain"`
}

func main() {
	var args Arguments
	arg.MustParse(&args)

	var printer rippler.ReportPrinter

	switch args.OutputFormat {
	case "plain":
		printer = printers.NewPlainPrinter()
	case "json":
		printer = printers.NewJSONPrinter()
	case "test-plan":
		printer = printers.NewTestPlanPrinter()
	case "test-matrix":
		printer = printers.NewTestMatrixPrinter()
	case "explain":
		printer = printers.NewExplainPrinter()
	default:
		log.Fatalf("Invalid output format: %s. Valid options are: plain, json, test-plan, test-matrix, explain", args.OutputFormat)
	}

	rip, err := rippler.NewRippler(args.Base, args.Path)
	if err != nil {
		log.Fatalf("Failed to initialize rippler: %v\n", err)
	}

	report, err := rip.Changes(context.TODO())
	if err != nil {
		log.Fatalf("Failed to get changes: %v\n", err)
	}

	if pErr := printer.Print(report); pErr != nil {
		log.Fatalf("Failed to print report: %v\n", pErr)
	}
}
