# go-ripple
 This tool analyzes a Go project and determines which packages are affected by recent changes.
 It is designed to be used in CI pipelines or development workflows where efficient testing or linting
 of only the impacted packages is desired.

 ## Features:

 - Detects all .go files that have changed compared to a base branch or commit (default: origin/main).
 - Maps changed files to their corresponding Go packages.
 - Propagates affected status to packages that import the changed packages (recursively).
 - Detects if "go.mod" has changed and, if so:
   - Parses the previous and current versions of go.mod.
   - Compares dependencies (modules) and identifies which ones were added, removed, or had version changes.
   - Identifies project packages that import the changed modules (transitively).

 - Outputs the list of all affected packages in various formats:
   - Plain text (one package per line).
   - JSON array of affected packages.
   - JSON plan format that groups affected packages by application (if applicable) and lists others separately.
 ## Installation:

`go install github.com/tangelo-labs/go-ripple@latest`

 Ensure you have Go installed (version 1.16 or later recommended).
 ## Usage:
                
`go-ripple [-b <base>] [-o <output>]`

 #### Example:

`go-ripple -b origin/main -o json`

 Dependencies:

 - Git must be installed and accessible via the system PATH.
 - The project must have a valid go.mod and go.sum at its root.
 - The base reference (e.g. origin/main) must be fetchable by Git.

 ### Argument Flags:

 `-b, --base `  The Git base branch or commit to compare against. Defaults to "origin/main".

 `-o, --output` The output format: "json", "plain" or "explain".
 
This script is intended for monorepos or large Go projects where full builds or tests
 are expensive and should be scoped to only affected components.