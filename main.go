package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/PIRSON21/generator-kafka-client/internal/generator"
	"github.com/PIRSON21/generator-kafka-client/internal/parser"
)

func main() {
	schema := flag.String("schema", "", "path to schema.go (required)")
	outDir := flag.String("out", "", "output directory for generated files (required)")
	packageName := flag.String("package", "", "Go package name for generated files (required)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: kafkagen -schema=<path> -out=<dir> -package=<name>\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	var missing []string
	if *schema == "" {
		missing = append(missing, "-schema")
	}
	if *outDir == "" {
		missing = append(missing, "-out")
	}
	if *packageName == "" {
		missing = append(missing, "-package")
	}
	if len(missing) > 0 {
		for _, f := range missing {
			fmt.Fprintf(os.Stderr, "error: flag %s is required\n", f)
		}
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(1)
	}

	def, err := parser.Parse(*schema)
	if err != nil {
		log.Fatalf("parse schema: %v", err)
	}

	def.PackageName = *packageName

	if err := generator.Generate(def, *outDir); err != nil {
		log.Fatalf("generate: %v", err)
	}

	fmt.Printf("Generated in %s:\n", *outDir)
	fmt.Println("  types.go")
	fmt.Println("  client.go")
	fmt.Println("  server.go")
	fmt.Println("  kafka/")
}
