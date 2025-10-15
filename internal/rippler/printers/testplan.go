package printers

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tangelo-labs/go-ripple/internal/model"
	"github.com/tangelo-labs/go-ripple/internal/rippler"
)

type plan struct {
	Apps   map[string][]string `json:"apps"`
	Others []string            `json:"others"`
}

type testPlan struct{}

// NewTestPlanPrinter creates a new instance of the test plan printer.
// It formats the affected packages into a JSON format, and filters out
// indirect packages that are not part of the project module, so they cannot be tested.
func NewTestPlanPrinter() rippler.ReportPrinter {
	return &testPlan{}
}

func (j *testPlan) Print(report *rippler.Report) error {
	str, err := j.drawJSONPlan(report.AffectedPackages, report.GoMod.Module.Path)
	if err != nil {
		return fmt.Errorf("failed to draw JSON plan: %w", err)
	}

	fmt.Println(str)

	return nil
}

func (j *testPlan) drawJSONPlan(affectedChanges []model.AffectedPackage, modName string) (string, error) {
	jPlan, err := json.MarshalIndent(buildPlan(affectedChanges, modName), "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON plan: %w", err)
	}

	return string(jPlan), nil
}

func buildPlan(affectedChanges []model.AffectedPackage, modName string) plan {
	affected := make(map[string]model.AffectedPackage)
	for _, pkg := range affectedChanges {
		affected[pkg.ImportPath] = pkg
	}

	result := plan{
		Apps:   make(map[string][]string),
		Others: make([]string, 0),
	}

	appsPrefix := modName + "/apps/"

	for _, pkg := range affected {
		if pkg.Indirect {
			// skip packages that are not part of the project module.
			continue
		}

		if !strings.HasPrefix(pkg.ImportPath, appsPrefix) {
			result.Others = append(result.Others, pkg.ImportPath)

			continue
		}

		appName := strings.Split(strings.TrimPrefix(pkg.ImportPath, appsPrefix), "/")[0]
		if _, ok := result.Apps[appName]; !ok {
			result.Apps[appName] = []string{}
		}

		result.Apps[appName] = append(result.Apps[appName], pkg.ImportPath)
	}

	return result
}
