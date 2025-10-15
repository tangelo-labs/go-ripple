package printers

import (
	"encoding/json"
	"fmt"

	"github.com/tangelo-labs/go-ripple/internal/rippler"
)

type jsonPrinter struct{}

// NewJSONPrinter creates a new instance of the JSON printer for displaying ripple reports.
func NewJSONPrinter() rippler.ReportPrinter {
	return &jsonPrinter{}
}

func (j *jsonPrinter) Print(report *rippler.Report) error {
	jsonData, err := json.MarshalIndent(report.AffectedPackages, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	fmt.Println(string(jsonData))

	return nil
}
