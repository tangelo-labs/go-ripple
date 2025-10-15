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

func (p *plainPrinter) Print(report *rippler.Report) error {
	for i := range report.AffectedPackages {
		fmt.Println(report.AffectedPackages[i].ImportPath)
	}

	return nil
}
