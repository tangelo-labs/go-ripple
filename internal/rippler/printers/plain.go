package printers

import (
	"fmt"

	"github.com/tangelo-labs/go-ripple/internal/rippler"
)

type plainPrinter struct{}

// NewPlainPrinter creates a new instance of the plain printer for displaying ripple reports.
func NewPlainPrinter() rippler.ReportPrinter {
	return &plainPrinter{}
}

// Print prints the import paths of all affected packages, one per line.
func (p *plainPrinter) Print(report *rippler.Report) error {
	for i := range report.AffectedPackages {
		fmt.Println(report.AffectedPackages[i].ImportPath)
	}

	return nil
}
