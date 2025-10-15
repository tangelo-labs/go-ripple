package model

// Package represents the output of `go list -json ./...`.
type Package struct {
	// Dir is the absolute path to the package directory.
	Dir string

	// GoFiles are the Go source files in the package, relative to Dir.
	GoFiles []string

	// ImportPath is the import path of the package, e.g. "github.com/me/project/users".
	ImportPath string

	// Imports is the list of import paths used by this package.
	Imports []string

	// TestImports are the import paths used by the package's (internal) test files.
	TestImports []string

	// XTestImports are the import paths used by the package's (external) test files.
	XTestImports []string

	// Deps are the module dependencies of the package.
	Deps []string
}

// AffectedPackage represents a package that is affected by a change.
type AffectedPackage struct {
	// ImportPath is the import path of the package, e.g. "github.com/me/project/users".
	ImportPath string

	// Indirect indicates whether the package is an indirect dependency.
	Indirect bool
}
