package analyzer

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/touillad/codeanalyzer/utils"
)

type Analyzer interface {
	Analyze() (map[string]*NodeInfo, error)
}

type analyzer struct {
	PackageName string
	IgnoreNodes []string
}

type Option func(a *analyzer)

func NewAnalyzer(packageName string, options ...Option) Analyzer {
	analyzer := &analyzer{
		PackageName: packageName,
	}

	for _, option := range options {
		option(analyzer)
	}

	return analyzer
}

func WithIgnoreList(files ...string) Option {
	return func(a *analyzer) {
		a.IgnoreNodes = files
	}
}

func (p *analyzer) IsInvalidPath(path string) bool {
	for _, value := range p.IgnoreNodes {
		return strings.Contains(path, value)
	}

	return false
}

func (a *analyzer) Analyze() (map[string]*NodeInfo, error) {
	summary := make(map[string]*NodeInfo)
	root := fmt.Sprintf("%s/src/%s", os.Getenv("GOPATH"), a.PackageName)
	fmt.Printf("rootPath is %s\n", root)
	err := filepath.Walk(root, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return fmt.Errorf("error on file walk: %s", err)
		}

		fileSet := token.NewFileSet()
		if f.IsDir() || !utils.IsGoFile(f.Name()) || a.IsInvalidPath(path) {
			return nil
		}

		file, err := parser.ParseFile(fileSet, path, nil, parser.ParseComments)
		if err != nil {
			// TODO: Log error with project information
			log.Printf("invalid file %s: %s", path, err)
			return nil
		}

		v := &Visitor{
			FileSet:     fileSet,
			PackageName: a.PackageName,
			Path:        path,
			StructInfo:  summary,
		}

		ast.Walk(v, file)
		if err != nil {
			return fmt.Errorf("error on walk: %s", err)
		}

		return nil
	})

	return summary, err
}
