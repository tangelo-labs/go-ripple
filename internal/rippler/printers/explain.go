package printers

import (
	"fmt"

	"github.com/tangelo-labs/go-ripple/internal/rippler"
)

type explainPrinter struct{}

type treeNode struct {
	PackageName string
	Children    []*treeNode
}

// NewExplainPrinter creates a new instance of the explain printer for displaying ripple reports.
func NewExplainPrinter() rippler.ReportPrinter {
	return &explainPrinter{}
}

func (p *explainPrinter) Print(report *rippler.Report) error {
	fmt.Println("Direct changes detected:")
	p.changes(report)

	println()
	println()

	fmt.Println("Dependency tree of affected packages:")
	p.tree(report)

	return nil
}

func (p *explainPrinter) changes(report *rippler.Report) {
	if len(report.Changes) == 0 {
		return
	}

	for i := range report.Changes {
		fmt.Printf("- %s\n", report.Changes[i].PackageName)

		if len(report.Changes[i].Reasons) > 0 {
			fmt.Println("  Reasons:")

			for _, reason := range report.Changes[i].Reasons {
				fmt.Printf("  - %s\n", reason)
			}
		} else {
			fmt.Println("  No specific reasons provided for this change.")
		}
	}
}

func (p *explainPrinter) tree(report *rippler.Report) {
	roots := p.buildTree(report)

	if len(roots) == 0 {
		return
	}

	directChangedPackages := make(map[string]struct{})
	for i := range report.Changes {
		directChangedPackages[report.Changes[i].PackageName] = struct{}{}
	}

	for i, root := range roots {
		p.printTreeNode(root, "", i == len(roots)-1, directChangedPackages)
	}
}

func (p *explainPrinter) printTreeNode(node *treeNode, prefix string, isLast bool, highlight map[string]struct{}) {
	treeSymbol := "├──"
	if isLast {
		treeSymbol = "└──"
	}

	packageName := node.PackageName
	if _, isDirectChange := highlight[node.PackageName]; isDirectChange {
		packageName = fmt.Sprintf("\033[32m%s\033[0m", packageName) // ANSI escape code for green
	}

	fmt.Printf("%s%s %s\n", prefix, treeSymbol, packageName)

	childPrefix := prefix

	appendix := "│   "
	if isLast {
		appendix = "    "
	}

	childPrefix += appendix

	for i, child := range node.Children {
		p.printTreeNode(child, childPrefix, i == len(node.Children)-1, highlight)
	}
}

func (p *explainPrinter) buildTree(report *rippler.Report) []*treeNode {
	dependencyMap := make(map[string][]string)

	for i := range report.AllPackages {
		allImports := append(append(report.AllPackages[i].Imports, report.AllPackages[i].TestImports...), report.AllPackages[i].XTestImports...)

		for _, imported := range allImports {
			dependencyMap[imported] = append(dependencyMap[imported], report.AllPackages[i].ImportPath)
		}
	}

	// cycles detector.
	visited := make(map[string]bool)

	var buildTreeNode func(pkg string) *treeNode

	buildTreeNode = func(pkg string) *treeNode {
		if visited[pkg] {
			return nil
		}

		visited[pkg] = true
		node := &treeNode{PackageName: pkg}

		for _, dependent := range dependencyMap[pkg] {
			if child := buildTreeNode(dependent); child != nil {
				node.Children = append(node.Children, child)
			}
		}

		return node
	}

	var roots []*treeNode

	for i := range report.Changes {
		if root := buildTreeNode(report.Changes[i].PackageName); root != nil {
			roots = append(roots, root)
		}
	}

	return roots
}
