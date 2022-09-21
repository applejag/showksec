package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

var flags struct {
	showHelp bool
}

func init() {
	pflag.BoolVarP(&flags.showHelp, "help", "h", false, "Show this help text")

	pflag.Usage = func() {
		fmt.Fprintf(os.Stderr, `Usage: showksec [file]

Base64-decodes secrets in Kubernetes secrets.
Will read from STDIN by default, but can also read from a given file.

Flags:
`)
		pflag.PrintDefaults()
	}
}

func main() {
	pflag.Parse()
	if flags.showHelp {
		pflag.Usage()
		return
	}

	readCloser := os.Stdin
	if pflag.NArg() > 0 {
		path := pflag.Arg(0)
		file, err := os.Open(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "showksec: error opening file: %s\n", err)
			os.Exit(1)
		}
		readCloser = file
	}

	docs := readAllDocs(readCloser)

	for _, root := range docs {
		for i := 0; i < len(root.Content); i += 2 {
			key := root.Content[i]
			value := root.Content[i+1]

			if key.Kind != yaml.ScalarNode || key.Value != "data" {
				continue
			}
			if value.Kind != yaml.MappingNode {
				continue
			}
			modifyDataNode(key, value)
		}
	}

	enc := yaml.NewEncoder(os.Stdout)
	enc.SetIndent(2)
	defer enc.Close()
	for _, doc := range docs {
		if err := enc.Encode(doc); err != nil {
			fmt.Fprintf(os.Stderr, "showksec: error encoding to stdout: %s\n", err)
			os.Exit(1)
		}
	}
}

func modifyDataNode(key, node *yaml.Node) {
	key.Value = "stringData"
	for i := 0; i < len(node.Content); i += 2 {
		value := node.Content[i+1]

		decoded, err := base64.StdEncoding.DecodeString(value.Value)
		if err != nil {
			value.LineComment = "base64 decode error: " + err.Error()
			value.Value = ""
			continue
		}
		value.Value = string(decoded)
	}
}

func readAllDocs(r io.ReadCloser) []yaml.Node {
	defer r.Close()
	dec := yaml.NewDecoder(r)
	var nodes []yaml.Node
	for {
		var node yaml.Node
		if err := dec.Decode(&node); err != nil {
			if errors.Is(err, io.EOF) {
				return nodes
			}
			fmt.Fprintf(os.Stderr, "showksec: error parsing stdin: %s\n", err)
			os.Exit(1)
		}
		nodes = append(nodes, *node.Content[0])
	}
}
