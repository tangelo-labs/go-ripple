package model

// GoMod represents the structure of a go.mod file in JSON format.
type GoMod struct {
	// Module is the main module declaration, which includes the module path and version.
	Module GoModDependency `json:"Module"`

	// Go specifies the Go version used by the module.
	Go string `json:"Go"`

	// Require lists the module dependencies required by this module.
	Require []GoModDependency `json:"Require"`

	// Exclude lists the module dependencies that are excluded from this module.
	Exclude []GoModDependency `json:"Exclude,omitempty"`

	// Replace lists the replacement directives for module dependencies.
	Replace []GoModReplace `json:"Replace"`

	// Tool lists the tool dependencies, which are typically used for development tools
	Tool []GoModDependency `json:"Tool,omitempty"`
}

// GoModDependency represents a dependency in the go.mod file, usually
// containing the module path and version. The `Indirect` field indicates
// whether the dependency is indirect (i.e., not directly imported by the module).
type GoModDependency struct {
	Path     string `json:"Path"`
	Version  string `json:"Version,omitempty"`
	Indirect bool   `json:"Indirect,omitempty"`
}

// GoModReplace represents a replacement directive in the go.mod file.
type GoModReplace struct {
	Old GoModDependency `json:"Old"`
	New GoModDependency `json:"New"`
}
