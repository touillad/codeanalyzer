package utils

import (
	"fmt"
	"os"
	"regexp"
	"strings"
)

var (
	regexpFile = regexp.MustCompile(`([^/]+\.go)(?:\.\((\w+)\))?$`)
)

func GetIdentifier(path, pkg, name string) string {
	path = TrimGoPath(path, pkg)
	if len(name) > 0 {
		return fmt.Sprintf("%s.(%s)", path, name)
	}
	return path
}

func TrimGoPath(path, repository string) string {
	return strings.TrimPrefix(path, fmt.Sprintf("%s/src/%s", os.Getenv("GOPATH"), repository))
}

func IsGoFile(name string) bool {
	return strings.HasSuffix(name, ".go")
}

func GetFileAndStruct(identifier string) (fileName, structName string) {
	result := regexpFile.FindStringSubmatch(identifier)
	if len(result) > 1 {
		fileName = result[1]
	}

	if len(result) > 2 {
		structName = result[2]
	}

	return
}
