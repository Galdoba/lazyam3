package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Galdoba/lazyam3/internal/infra/metadata"
)

func main() {
	// Flags
	id := flag.Int64("id", 0, "ID of the element to search for (required)")
	file := flag.String("file", "metadata.json", "path to the json file with metadata")
	shallow := flag.Bool("shallow", false, "do not include descendants of the target object (only it and its parents)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Search for a metadata element by id.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n")
		fmt.Fprintf(os.Stderr, "  %s -id=<number> [-file=<path>] [-shallow]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s -id=42\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -id=42 -file=./data/catalog.json\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -id=42 -shallow\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	// Verify that -id was actually provided (0 is a valid id value)
	idSet := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "id" {
			idSet = true
		}
	})
	if !idSet {
		fmt.Fprintln(os.Stderr, "error: required parameter -id is not set")
		fmt.Fprintln(os.Stderr)
		flag.Usage()
		os.Exit(2)
	}

	// Read the file
	data, err := os.ReadFile(*file)
	if err != nil {
		log.Fatalf("failed to read file %q: %v", *file, err)
	}

	// Deserialization
	var root metadata.Root
	if err := json.Unmarshal(data, &root); err != nil {
		log.Fatalf("failed to parse json from %q: %v", *file, err)
	}

	// Select mode
	descendants := metadata.WithDescendants
	if *shallow {
		descendants = metadata.WithoutDescendants
	}

	// Search
	found, ok := root.FindByID(*id, descendants)
	if !ok {
		fmt.Fprintf(os.Stderr, "element with id=%d not found\n", *id)
		os.Exit(1)
	}

	// Output
	out, err := json.MarshalIndent(found, "", "  ")
	if err != nil {
		log.Fatalf("failed to serialize result: %v", err)
	}
	fmt.Println(string(out))
}
