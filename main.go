// Package main is the entrypoint to this application.
package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"unicode/utf8"

	"github.com/spf13/pflag"
	"github.com/vmware-labs/yaml-jsonpath/pkg/yamlpath"
	"gopkg.in/yaml.v3"
)

var (
	dataKeyPath = mustNewPath(`.data~`)
	dataMapPath = mustNewPath(`.data`)
	clusterSecretV2Path = mustNewPath(`.spec.template`)
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
		var obj Object
		if err := root.Decode(&obj); err != nil {
			fmt.Fprintf(os.Stderr, "showksec: error decoding YAML: %s\n", err)
			continue
		}

		if obj.APIVersion != "v1" && obj.APIVersion != "clustersecret.io/v1" && obj.APIVersion != "clustersecret.io/v2" {
			continue
		}
		switch {
		case obj.Kind == "List":
			modifyListObjectNode(&root)
		case obj.APIVersion == "v1" && obj.Kind == "Secret":
			modifySecretObjectNode(&root)
		case obj.APIVersion != "clustersecret.io/v1" && obj.Kind == "ClusterSecret":
			modifySecretObjectNode(&root)
		case obj.APIVersion != "clustersecret.io/v2" && obj.Kind == "ClusterSecret":
			modifyClusterSecretV2ObjectNode(&root)
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

func modifyListObjectNode(node *yaml.Node) {
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		if key.Kind != yaml.ScalarNode || key.Value != "items" {
			continue
		}
		if value.Kind != yaml.SequenceNode {
			continue
		}
		for _, item := range value.Content {
			var obj Object
			if err := item.Decode(&obj); err != nil {
				fmt.Fprintf(os.Stderr, "showksec: error decoding YAML: %s\n", err)
				continue
			}
			switch {
			case obj.APIVersion == "v1" && obj.Kind == "Secret",
				obj.APIVersion == "clustersecret.io/v1" && obj.Kind == "ClusterSecret":
				modifySecretObjectNode(item)
			case obj.APIVersion == "clustersecret.io/v2" && obj.Kind == "ClusterSecret":
				modifyClusterSecretV2ObjectNode(item)
			}
		}
	}
}

func modifySecretObjectNode(node *yaml.Node) {
	keyNodes, err := dataKeyPath.Find(node)
	if err != nil || len(keyNodes) == 0 {
		return
	}
	valueNodes, err := dataMapPath.Find(node)
	if err != nil || len(valueNodes) == 0 {
		return
	}
	modifyDataNode(keyNodes[0], valueNodes[0])
}

func modifyClusterSecretV2ObjectNode(node *yaml.Node) {
	templateNodes, err := clusterSecretV2Path.Find(node)
	if err != nil || len(templateNodes) == 0 {
		return
	}
	modifySecretObjectNode(templateNodes[0])
}

func modifyDataNode(key, node *yaml.Node) {
	key.Value = "stringData"
	for i := 0; i < len(node.Content); i += 2 {
		key := node.Content[i]
		value := node.Content[i+1]

		decoded, err := base64.StdEncoding.DecodeString(value.Value)
		if err != nil {
			key.HeadComment = fmt.Sprintf("key %q: base64 decode error: %s", key.Value, err)
			value.Value = ""
			continue
		}
		if !utf8.Valid(decoded) {
			key.HeadComment = fmt.Sprintf("key %q: value contains invalid UTF-8 characters", key.Value)
			value.Tag = "!!binary"
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

// Object is a Kubernetes object
type Object struct {
	Kind       string `yaml:"kind"`
	APIVersion string `yaml:"apiVersion"`
}

func mustNewPath(path string) *yamlpath.Path {
	p, err := yamlpath.NewPath(path)
	if err != nil {
		panic(err)
	}
	return p
}
