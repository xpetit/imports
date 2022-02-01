package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

func check(a ...interface{}) {
	for _, v := range a {
		if err, ok := v.(error); ok && err != nil {
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

var (
	maxDepth int

	wg sync.WaitGroup

	waitlist = make(chan struct{}, runtime.NumCPU())

	m       sync.Mutex
	visited = map[string]struct{}{}
)

func printImports(depth int, base, target string) {
	defer wg.Done()

	if maxDepth > 0 && depth > maxDepth {
		return
	}

	m.Lock()
	if _, ok := visited[target]; ok {
		m.Unlock()
		return
	}
	visited[target] = struct{}{}
	m.Unlock()

	waitlist <- struct{}{}
	defer func() { <-waitlist }()

	imports := strings.Split(string(mustOutput("go", "list", "-f", `{{join .Imports "\n"}}`, target)), "\n")
	for _, pkg := range imports {
		if !strings.HasPrefix(pkg, base) { // ignore external imports
			continue
		}
		relTarget, err := filepath.Rel(base, target)
		check(err)
		relPkg, err := filepath.Rel(base, pkg)
		check(err)
		fmt.Printf(`	"%s" -> "%s"`+"\n", relTarget, relPkg)
		wg.Add(1)
		go printImports(depth+1, base, pkg)
	}
}

func main() {
	flag.IntVar(&maxDepth, "depth", 0, "max depth, 0 means no limit")
	flag.Parse()
	if flag.NArg() > 0 {
		check(os.Chdir(flag.Arg(0)))
	}
	fields := strings.Fields(string(mustOutput("go", "list", "-f", `{{.Module.Path}} {{.ImportPath}}`)))
	base := fields[0]
	target := fields[1]

	fmt.Println("digraph {")
	fmt.Println("	rankdir=LR")
	fmt.Println("	node [shape=box]")
	wg.Add(1)
	go printImports(1, base, target)
	wg.Wait()
	fmt.Println("}")
}
