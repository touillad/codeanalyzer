package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"

	"github.com/touillad/codeanalyzer/analyzer"
	"github.com/touillad/codeanalyzer/model"
)

func main() {

	projectPath := flag.String("pp", ".", "project path")
	flag.Parse()
	//fmt.Printf("projectPath is %s\n", *projectPath)

	analyzer := analyzer.NewAnalyzer(*projectPath, analyzer.WithIgnoreList("/vendor/"))
	summary, err := analyzer.Analyze()
	if err != nil {
		fmt.Printf("Error: ", err)
	}

	body, err := json.Marshal(model.New(summary, *projectPath))
	if err != nil {
		fmt.Printf("Error: ", err)
	}
	//fmt.Printf("%v", &body)

	err = ioutil.WriteFile("/tmp/dat4", body, 0644)
	if err != nil {
		panic(err)
	}

}
