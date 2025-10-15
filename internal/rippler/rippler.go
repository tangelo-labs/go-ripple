package rippler

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/tangelo-labs/go-ripple/internal/model"
)

// Rippler is the main struct that handles the ripple detection logic.
type Rippler struct {
	goModFilePath string
	baseBranch    string
}

// Report holds the results of the ripple detection process.
type Report struct {
	// GoMod contains the parsed go.mod file.
	GoMod model.GoMod

	// DirtyFiles contains the list of Go files that have changed compared to the base branch.
	DirtyFiles []string

	// AllPackages contains the list of all packages in the Go project.
	AllPackages []model.Package

	// AffectedPackages contains the list of packages that are affected by the changes.
	// This includes packages that are directly affected by file changes,
	// changes in go.mod, and changes in indirect third-party modules. So it may include
	// third-party packages that are not directly managed by the project.
	AffectedPackages []model.AffectedPackage

	// Changes contains the list of detected changes in the Go project.
	Changes []Change
}

// AffectedPackage represents a package that is affected by changes.
type AffectedPackage struct {
	Path string
}

// Change represents a detected change in the Go project.
type Change struct {
	// PackageName is the name of the package that is affected by the change.
	PackageName string

	// Reasons is a list of reasons why this package is considered changed.
	// It can include file changes, go.mod changes, or external module changes.
	Reasons []string
}

// NewRippler creates a new instance of Rippler with the specified base branch.
func NewRippler(baseBranch string, modulePath string, opts ...Option) (*Rippler, error) {
	modPath, err := filepath.Abs(modulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for go.mod: %w", err)
	}

	modPath = filepath.Join(modPath, "go.mod")

	modPath, err = filepath.Abs(modPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get absolute path for go.mod: %w", err)
	}

	if _, stErr := os.Stat(modPath); os.IsNotExist(stErr) {
		return nil, fmt.Errorf("go.mod file does not exist at path: %s", modPath)
	}

	rip := &Rippler{
		goModFilePath: modPath,
		baseBranch:    baseBranch,
	}

	for _, opt := range opts {
		if oErr := opt(rip); oErr != nil {
			return nil, fmt.Errorf("failed to apply rippler option: %w", oErr)
		}
	}

	return rip, nil
}

// Changes detects the changes in the Go project based on the provided base branch.
func (r *Rippler) Changes(ctx context.Context) (*Report, error) {
	report := &Report{}

	mod, err := r.parseGoMod(ctx, r.goModFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to parse go.mod: %w", err)
	}

	report.GoMod = mod

	allPackages, err := r.listAllPackages(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list all packages: %w", err)
	}

	report.AllPackages = allPackages

	dirtyFiles, err := r.getChangedGoFiles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed Go files: %w", err)
	}

	report.DirtyFiles = dirtyFiles

	// Direct file changes are the primary source of ripple detection.
	changes := r.affectedPackagesByFileChanges(report)

	{
		affectedByModChange, aErr := r.affectedPackagesByGoModChange(ctx, report)
		if aErr != nil {
			return nil, fmt.Errorf("failed to determine affected packages by go.mod change: %w", aErr)
		}

		changes = append(changes, affectedByModChange...)
	}

	{
		affectedByByExternalModChange, aErr := r.affectedPackagesByExternalModule(ctx, report)
		if aErr != nil {
			return nil, fmt.Errorf("failed to determine affected packages by external module change: %w", aErr)
		}

		changes = append(changes, affectedByByExternalModChange...)
	}

	report.Changes = unifyChanges(changes)
	report.AffectedPackages = r.propagateAffectedPackages(report)

	return report, nil
}

func (r *Rippler) parseGoMod(ctx context.Context, path string) (model.GoMod, error) {
	cmd := exec.CommandContext(ctx, "go", "mod", "edit", "-json", path)

	out, err := cmd.Output()
	if err != nil {
		return model.GoMod{}, fmt.Errorf("failed to parse go.mod (%s): %w", path, err)
	}

	var mod model.GoMod
	if juErr := json.Unmarshal(out, &mod); juErr != nil {
		return model.GoMod{}, fmt.Errorf("failed to unmarshal go.mod: %w", juErr)
	}

	return mod, nil
}

func (r *Rippler) getChangedGoFiles(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", r.baseBranch)

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("git diff failed: %w", err)
	}

	goFiles := make([]string, 0)
	outLines := strings.Split(string(out), "\n")

	for i := range outLines {
		if !strings.HasSuffix(outLines[i], ".go") {
			continue
		}

		abs, fpErr := filepath.Abs(outLines[i])
		if fpErr != nil {
			return nil, fmt.Errorf("failed to get absolute path for %s: %w", outLines[i], fpErr)
		}

		goFiles = append(goFiles, abs)
	}

	return goFiles, nil
}

func (r *Rippler) listAllPackages(ctx context.Context) ([]model.Package, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-json", "./...")
	out := bytes.Buffer{}
	cmd.Stdout = &out
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("go list failed: %w", err)
	}

	var packages []model.Package

	decoder := json.NewDecoder(&out)
	for decoder.More() {
		var pkg model.Package

		if err := decoder.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("failed to decode package: %w", err)
		}

		packages = append(packages, pkg)
	}

	return packages, nil
}

// affectedPackagesByFileChanges determines which packages are affected by the changes in dirty files.
func (r *Rippler) affectedPackagesByFileChanges(report *Report) []Change {
	affected := make(map[string]Change)
	pkgMap := r.mapPackagesByFile(report.AllPackages)

	for i := range report.DirtyFiles {
		if pkg, ok := pkgMap[report.DirtyFiles[i]]; ok {
			if _, exists := affected[pkg]; !exists {
				affected[pkg] = Change{
					PackageName: pkg,
					Reasons: []string{
						fmt.Sprintf("file %s has changed", report.DirtyFiles[i]),
					},
				}
			} else {
				ch := affected[pkg]
				ch.Reasons = append(ch.Reasons, fmt.Sprintf("file %s has changed", report.DirtyFiles[i]))
				affected[pkg] = ch
			}
		}
	}

	out := make([]Change, 0)
	for i := range affected {
		out = append(out, affected[i])
	}

	return out
}

// mapPackagesByFile creates a mapping from absolute file paths to their corresponding package import paths.
func (r *Rippler) mapPackagesByFile(pkgs []model.Package) map[string]string {
	result := make(map[string]string)

	for i := range pkgs {
		for j := range pkgs[i].GoFiles {
			fullPath := filepath.Join(pkgs[i].Dir, pkgs[i].GoFiles[j])
			result[fullPath] = pkgs[i].ImportPath
		}
	}

	return result
}

// affectedPackagesByGoModChange determines which packages are affected by changes in go.mod.
// It checks if the go.mod file has changed compared to the base branch and identifies affected
// packages based on module changes.
//
// For example, if a new module was added/removed or an existing module's version was changed.
// This method collects all those modules, so it can later determine which packages
// depend on those modules and thus are affected by the change in go.mod.
func (r *Rippler) affectedPackagesByGoModChange(ctx context.Context, report *Report) ([]Change, error) {
	affected := make([]Change, 0)

	modChanged, err := r.goModHasChanged(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check if go.mod has changed: %w", err)
	}

	if !modChanged {
		return nil, nil
	}

	changedMods, cmErr := r.getChangedModules(ctx, report.GoMod)
	if cmErr != nil {
		return nil, fmt.Errorf("failed to get changed modules: %w", cmErr)
	}

	for _, mod := range changedMods {
		affected = append(affected, Change{
			PackageName: mod,
			Reasons: []string{
				fmt.Sprintf("module %s has changed in go.mod", mod),
			},
		})
	}

	return affected, nil
}

// affectedPackagesByExternalModule determines which packages are affected by changes in indirect third-party modules.
// It is similar to affectedPackagesByGoModChange, but specifically focuses on modules that are not directly
// managed by the project (i.e., indirect dependencies). The main difference is that it also takes into account
// changes in the go.sum file, which may affect the resolution of indirect dependencies.
//
// For example, you project may depend on a module that has not changed, but that module may depend on another module
// that has changed. In such cases, the indirect module may have been updated in the go.sum file, which can affect
// the resolution of the indirect dependencies. This method collects all those indirect modules, so it can later
// determine which packages depend on those modules and thus are affected by the change in go.sum.
func (r *Rippler) affectedPackagesByExternalModule(ctx context.Context, _ *Report) ([]Change, error) {
	affected := make([]Change, 0)

	indirectMods, err := r.getChangedIndirectModules(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get changed indirect modules: %w", err)
	}

	for _, mod := range indirectMods {
		affected = append(affected, Change{
			PackageName: mod,
			Reasons: []string{
				fmt.Sprintf("indirect module %s has changed in go.sum", mod),
			},
		})
	}

	return affected, nil
}

func (r *Rippler) goModHasChanged(ctx context.Context) (bool, error) {
	cmd := exec.CommandContext(ctx, "git", "diff", "--name-only", r.baseBranch, "--", "go.mod")

	out, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("git diff for go.mod failed: %w", err)
	}

	return strings.TrimSpace(string(out)) != "", nil
}

func (r *Rippler) getChangedModules(ctx context.Context, currentGoMod model.GoMod) ([]string, error) {
	tmp := filepath.Join(os.TempDir(), "go.mod.base")
	cmd := exec.CommandContext(ctx, "git", "show", r.baseBranch+":go.mod")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get base go.mod: %w", err)
	}

	if wfErr := os.WriteFile(tmp, out, 0644); wfErr != nil {
		return nil, fmt.Errorf("failed to write temp go.mod: %w", wfErr)
	}

	oldMod, err := r.parseGoMod(ctx, tmp)
	if err != nil {
		return nil, err
	}

	oldSet := make(map[string]string)

	for i := range oldMod.Require {
		oldSet[oldMod.Require[i].Path] = oldMod.Require[i].Version
	}

	var changed []string

	for i := range currentGoMod.Require {
		if oldVer, ok := oldSet[currentGoMod.Require[i].Path]; !ok || oldVer != currentGoMod.Require[i].Version {
			changed = append(changed, currentGoMod.Require[i].Path)
		}
	}

	return changed, nil
}

func (r *Rippler) getChangedIndirectModules(ctx context.Context) ([]string, error) {
	baseMods, err := r.getBaseModules(ctx)
	if err != nil {
		return nil, err
	}

	currentMods, err := r.getAllModules(ctx)
	if err != nil {
		return nil, err
	}

	var changed []string

	for path, newVer := range currentMods {
		oldVer, exists := baseMods[path]
		if !exists || oldVer != newVer {
			changed = append(changed, path)
		}
	}

	return changed, nil
}

func (r *Rippler) getAllModules(ctx context.Context) (map[string]string, error) {
	cmd := exec.CommandContext(ctx, "go", "list", "-m", "all")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list current modules: %w", err)
	}

	modules := make(map[string]string)
	lines := strings.Split(string(out), "\n")

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			modules[fields[0]] = fields[1]
		}
	}

	return modules, nil
}

func (r *Rippler) getBaseModules(ctx context.Context) (map[string]string, error) {
	tmpMod := filepath.Join(os.TempDir(), "go.base.mod")
	tmpSum := filepath.Join(os.TempDir(), "go.base.sum")

	cmd := exec.CommandContext(ctx, "git", "show", r.baseBranch+":go.mod")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get base go.mod: %w", err)
	}

	if wfErr := os.WriteFile(tmpMod, out, 0644); wfErr != nil {
		return nil, fmt.Errorf("failed to write base go.mod: %w", wfErr)
	}

	cmd = exec.CommandContext(ctx, "git", "show", r.baseBranch+":go.sum")

	out, err = cmd.Output()
	if err == nil {
		if wfErr := os.WriteFile(tmpSum, out, 0644); wfErr != nil {
			return nil, fmt.Errorf("failed to write base go.sum: %w", wfErr)
		}
	}

	cmd = exec.CommandContext(ctx, "go", "list", "-m", "-modfile="+tmpMod, "all")

	out, err = cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list base modules: %w", err)
	}

	modules := make(map[string]string)
	lines := strings.Split(string(out), "\n")

	for i := range lines {
		fields := strings.Fields(lines[i])
		if len(fields) >= 2 {
			modules[fields[0]] = fields[1]
		}
	}

	return modules, nil
}

func (r *Rippler) propagateAffectedPackages(report *Report) []model.AffectedPackage {
	initial := report.Changes
	dependents := make(map[string][]string)
	initialMap := make(map[string]struct{})

	for i := range initial {
		initialMap[initial[i].PackageName] = struct{}{}
	}

	for i := range report.AllPackages {
		fullPackageImports := append(append(report.AllPackages[i].Imports, report.AllPackages[i].TestImports...), report.AllPackages[i].XTestImports...)
		for j := range fullPackageImports {
			dependents[fullPackageImports[j]] = append(dependents[fullPackageImports[j]], report.AllPackages[i].ImportPath)
		}
	}

	queue := make([]string, 0)
	for pkg := range initialMap {
		queue = append(queue, pkg)
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		for _, dep := range dependents[current] {
			if _, ok := initialMap[dep]; !ok {
				initialMap[dep] = struct{}{}
				queue = append(queue, dep)
			}
		}
	}

	out := make([]model.AffectedPackage, 0)
	for pkg := range initialMap {
		out = append(out, model.AffectedPackage{
			ImportPath: pkg,
			Indirect:   !strings.HasPrefix(pkg, report.GoMod.Module.Path),
		})
	}

	slices.SortFunc(out, func(a, b model.AffectedPackage) int {
		return strings.Compare(a.ImportPath, b.ImportPath)
	})

	return out
}

func unifyChanges(ch []Change) []Change {
	unified := make(map[string]Change)

	for i := range ch {
		if existing, exists := unified[ch[i].PackageName]; exists {
			existing.Reasons = append(existing.Reasons, ch[i].Reasons...)
			unified[ch[i].PackageName] = existing
		} else {
			unified[ch[i].PackageName] = ch[i]
		}
	}

	result := make([]Change, 0, len(unified))
	for i := range unified {
		result = append(result, unified[i])
	}

	return result
}
