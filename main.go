package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"

	"gopkg.in/yaml.v3"
)

func main() {
	docs := readAllDocs(os.Stdin)

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

func readAllDocs(r io.Reader) []yaml.Node {
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
