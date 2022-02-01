package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

func check(a ...interface{}) {
	for _, a := range a {
		if err, ok := a.(error); ok && err != nil {
			log.Fatal(err)
		}
	}
}

func mustOutput(name string, arg ...string) []byte {
	b, err := exec.Command(name, arg...).CombinedOutput()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			log.Fatal(string(b))
		} else {
			log.Fatal(err)
		}
	}
	return b
}

var wg sync.WaitGroup

func printImports(base, target string) {
	var Data struct {
		Imports []string
	}
	json.Unmarshal(mustOutput("go", "list", "-json", target), &Data)
	for _, pkg := range Data.Imports {
		if !strings.HasPrefix(pkg, base) { // ignore external imports
			continue
		}
		relTarget, err := filepath.Rel(base, target)
		check(err)
		relPkg, err := filepath.Rel(base, pkg)
		check(err)
		fmt.Printf(`	"%s" -> "%s"`+"\n", relTarget, relPkg)
		wg.Add(1)
		go printImports(base, pkg)
	}
	wg.Done()
}

func main() {
	if len(os.Args) > 1 {
		check(os.Chdir(os.Args[1]))
	}
	var Data struct {
		ImportPath string
		Module     struct {
			Path string
		}
	}
	check(json.Unmarshal(mustOutput("go", "list", "-json"), &Data))
	fmt.Println("digraph {")
	base := Data.Module.Path
	target := Data.ImportPath
	wg.Add(1)
	printImports(base, target)
	wg.Wait()
	fmt.Println("}")
}
