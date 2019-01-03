package model

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/touillad/codeanalyzer/analyzer"
	"github.com/touillad/codeanalyzer/utils"
)

type NodeType string

const (
	StructType  NodeType = "STRUCT"
	FileType    NodeType = "FILE"
	PackageType NodeType = "PACKAGE"
)

type Node struct {
	Name               string   `json:"name"`
	URL                string   `json:"url"`
	Type               NodeType `json:"type"`
	Width              float64  `json:"width"`
	Depth              float64  `json:"depth"`
	Position           Position `json:"position"`
	NumberOfLines      int      `json:"numberOfLines"`
	NumberOfMethods    int      `json:"numberOfMethods"`
	NumberOfAttributes int      `json:"numberOfAttributes"`
	NumberOfStmts      int
	Children           []*Node `json:"children"`
	Line               int     `json:"-"`
	childrenMap        map[string]*Node
}

const (
	BaseTypeFlag       = "{{TYPE}}"
	PackageBaseTypeURL = "tree"
	FileBaseTypeURL    = "blob"
)

func getNodeURL(node *Node, parentPath string) (raw string, formatted string) {
	if node.Type == StructType {
		formatted = fmt.Sprintf("%s#L%d", strings.Replace(parentPath, BaseTypeFlag, FileBaseTypeURL, -1), node.Line)
		return formatted, formatted
	}

	if len(node.Name) > 0 {
		raw = fmt.Sprintf("%s/%s", parentPath, node.Name)
	} else {
		raw = parentPath
	}

	if node.Type == PackageType {
		formatted = strings.Replace(raw, BaseTypeFlag, PackageBaseTypeURL, -1)
		return
	}
	formatted = strings.Replace(raw, BaseTypeFlag, FileBaseTypeURL, -1)
	return
}

func (n *Node) GenerateChildList(parentPath string) {
	for _, child := range n.childrenMap {
		baseName, nodeURL := getNodeURL(child, parentPath)
		child.URL = nodeURL
		n.Children = append(n.Children, child)
		if len(child.childrenMap) > 0 {
			child.GenerateChildList(baseName)
		}
	}

	// Sort by width
	sort.Sort(sort.Reverse(byWidth(n.Children)))
}

func (n *Node) GenerateChildrenPosition() {
	if len(n.Children) == 0 {
		n.Width = float64(n.NumberOfAttributes) + 1
		n.Depth = float64(n.NumberOfAttributes) + 1
		return
	}

	positionGenerator := NewGenerator(len(n.Children))
	for _, child := range n.Children {
		child.GenerateChildrenPosition()
		child.Position = positionGenerator.NextPosition(child.Width, child.Depth)
	}

	bounds := positionGenerator.GetBounds()
	n.Width, n.Depth = bounds.X, bounds.Y

	for _, child := range n.Children {
		child.Position.X -= n.Width / 2.0
		child.Position.Y -= n.Depth / 2.0
	}

	if n.Type == FileType {
		n.Width += float64(n.NumberOfAttributes)
		n.Depth += float64(n.NumberOfAttributes)
	}
}

func generateStateByComponents(rootPackageName, currentPackageName string, n *Node, stmtCount *int, components map[string]*Component) {
	if n == nil {
		return
	}

	if n.Type == PackageType {
		if n.Name != "" {
			if currentPackageName != "" {
				currentPackageName = currentPackageName + "." + n.Name
			} else {
				currentPackageName = n.Name
			}
		}
		_, ok := components[currentPackageName]
		if !ok {
			components[currentPackageName] = &Component{
				name:      currentPackageName,
				isRoot:    currentPackageName == rootPackageName,
				nbMethods: n.NumberOfMethods,
				nbFiles:   0,
			}
		}
	}

	cmp, ok := components[currentPackageName]
	if ok {
		if n.Type == FileType {
			cmp.nbFiles += 1
		}
		cmp.nbMethods += n.NumberOfMethods
		cmp.nbStmts += n.NumberOfStmts
		*stmtCount += n.NumberOfStmts
	}

	if len(n.Children) > 0 {
		for _, child := range n.Children {
			generateStateByComponents(rootPackageName, currentPackageName, child, stmtCount, components)
		}
	}

}

func getPathAndFile(fullPath string) (paths []string, fileName, structName string) {
	pathlist := strings.Split(fullPath, "/")
	paths = pathlist[:len(pathlist)-1]
	fileName, structName = utils.GetFileAndStruct(pathlist[len(pathlist)-1])
	return
}

func New(items map[string]*analyzer.NodeInfo, repositoryName string) *Node {
	tree := &Node{
		Name:        repositoryName,
		childrenMap: make(map[string]*Node),
		Children:    make([]*Node, 0),
	}

	for key, value := range items {
		currentNode := tree
		paths, fileName, structName := getPathAndFile(key)
		for _, path := range paths {
			_, ok := currentNode.childrenMap[path]
			if !ok {
				currentNode.childrenMap[path] = &Node{
					Name:        path,
					Type:        PackageType,
					childrenMap: make(map[string]*Node),
				}
			}
			currentNode = currentNode.childrenMap[path]
		}

		_, ok := currentNode.childrenMap[fileName]
		if !ok {
			currentNode.childrenMap[fileName] = &Node{
				Name:        fileName,
				Type:        FileType,
				childrenMap: make(map[string]*Node),
			}
		}

		fileNode := currentNode.childrenMap[fileName]

		if len(structName) > 0 {
			structNode, ok := fileNode.childrenMap[structName]
			if !ok {
				fileNode.childrenMap[structName] = &Node{
					Name:               structName,
					Type:               StructType,
					Line:               value.Line,
					NumberOfAttributes: value.NumberAttributes,
					NumberOfMethods:    value.NumberMethods,
					NumberOfLines:      value.NumberLines,
					NumberOfStmts:      value.NumberStmts,
				}
			} else {
				structNode.NumberOfAttributes += value.NumberAttributes
				structNode.NumberOfLines += value.NumberLines
				structNode.NumberOfMethods += value.NumberMethods
				structNode.NumberOfStmts += value.NumberStmts
			}
		} else {
			fileNode.NumberOfAttributes += value.NumberAttributes
			fileNode.NumberOfLines += value.NumberLines
			fileNode.NumberOfMethods += value.NumberMethods
			fileNode.NumberOfStmts += value.NumberStmts
		}
	}

	// TODO: branch selector
	tree.GenerateChildList(fmt.Sprintf("https://%s/%s/master", repositoryName, BaseTypeFlag))
	tree.GenerateChildrenPosition()

	nbTotalComponents := 0
	nbTotalStmts := 0
	components := make(map[string]*Component)
	//packageName := ""
	//tree.GenerateStats(packageName, &nbTotalComponents, &nbTotalMethods, components)

	generateStateByComponents(repositoryName, repositoryName, tree, &nbTotalStmts, components)
	nbTotalComponents = len(components)

	mean := float64(nbTotalStmts / nbTotalComponents)
	squareDiffSum := 0.0
	var keys []string
	for key, component := range components {
		//for _, component := range components {
		squareDiffSum += math.Pow(math.Abs(float64(component.nbStmts)-mean), 2.0)
		keys = append(keys, key)
	}
	sort.Strings(keys)
	stdDev := float32(math.Sqrt((squareDiffSum / float64(nbTotalComponents))))

	fmt.Printf("\nComponent Size Analysis\n")
	fmt.Printf("Service Name: %s\n", repositoryName)
	fmt.Printf("Total Statements: %d\n", nbTotalStmts)
	fmt.Printf("Total Components: %d\n", nbTotalComponents)
	fmt.Printf("Avg Statements Per Component: %.2f\n", mean)
	fmt.Printf("Standard Deviation: %.2f\n", stdDev)
	fmt.Println("")
	fmt.Println("%      stmts  files   sdev  root  component")

	for _, componentName := range keys {
		//for componentName, component := range components {
		component := components[componentName]
		percentage := float32(component.nbStmts) / float32(nbTotalStmts) * 100
		//if percentage == 0 {
		//	percentage = 1
		//}
		diffFromMean := math.Abs(float64(component.nbStmts) - mean)
		if component.nbStmts == 0 {
			diffFromMean = 0
		}
		numStdDev := diffFromMean / float64(stdDev)
		root := " "
		if component.isRoot {
			root = "*"
		}
		fmt.Printf("%-8.2f", percentage)
		fmt.Printf("%-8d", component.nbStmts)
		fmt.Printf("%-6d", component.nbFiles)
		fmt.Printf("%-6.2f", numStdDev)
		fmt.Printf("%-6s", root)
		fmt.Printf("%s\n", componentName)
	}

	return tree
}
