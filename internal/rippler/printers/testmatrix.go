package printers

import (
	"encoding/json"
	"fmt"

	"github.com/tangelo-labs/go-ripple/internal/model"
	"github.com/tangelo-labs/go-ripple/internal/rippler"
)

type matrixRow struct {
	Name     string   `json:"name"`
	IsApp    bool     `json:"is_app"`
	Packages []string `json:"packages"`
}

type testMatrix struct{}

// NewTestMatrixPrinter creates a new instance of the test matrix printer.
// It formats the affected packages into a JSON matrix format, and filters out
// indirect packages that are not part of the project module, so they cannot be tested.
func NewTestMatrixPrinter() rippler.ReportPrinter {
	return &testMatrix{}
}

func (j *testMatrix) Print(report *rippler.Report) error {
	str, err := j.drawJSONMatrix(report.AffectedPackages, report.GoMod.Module.Path)
	if err != nil {
		return fmt.Errorf("failed to draw JSON plan: %w", err)
	}

	fmt.Println(str)

	return nil
}

func (j *testMatrix) drawJSONMatrix(affectedChanges []model.AffectedPackage, modName string) (string, error) {
	result := buildPlan(affectedChanges, modName)
	matrix := make([]matrixRow, 0)

	for appName, pkgs := range result.Apps {
		matrix = append(matrix, matrixRow{
			Name:     appName,
			IsApp:    true,
			Packages: pkgs,
		})
	}

	if len(result.Others) > 0 {
		matrix = append(matrix, matrixRow{
			Name:     "@others",
			IsApp:    false,
			Packages: result.Others,
		})
	}

	jMatrix, err := json.MarshalIndent(matrix, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON matrix: %w", err)
	}

	return string(jMatrix), nil
}
