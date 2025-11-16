//go:build tools

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	const outFilename = "coverage.out"

	_ = os.Remove(outFilename)

	outFile, err := os.Create(outFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create %s: %v\n", outFilename, err)
		os.Exit(1)
	}
	defer outFile.Close()

	if _, err := outFile.WriteString("mode: set\n"); err != nil {
		fmt.Fprintf(os.Stderr, "failed to write mode to %s: %v\n", outFilename, err)
		os.Exit(1)
	}

	files, err := filepath.Glob("*.cover")
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find .cover files: %v\n", err)
		os.Exit(1)
	}

	if len(files) == 0 {
		fmt.Fprintln(os.Stderr, "warning: no .cover files found")
		return
	}

	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to read %s: %v\n", file, err)
			continue
		}

		lines := strings.Split(string(content), "\n")
		for i := 1; i < len(lines); i++ {
			if len(strings.TrimSpace(lines[i])) > 0 {
				if _, err := outFile.WriteString(lines[i] + "\n"); err != nil {
					fmt.Fprintf(os.Stderr, "failed to write to %s: %v\n", outFilename, err)
					os.Exit(1)
				}
			}
		}
	}
}
