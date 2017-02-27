package main

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/types"
	"golang.org/x/tools/go/loader"
	"reflect"
	"strings"
	"text/template"
)

// GenerateSchemaDefinitions returns schema definitions for all struct types
// found in the package at `path` including all fields that have a tag with name
// `tag`.
func GenerateSchemaDefinitions(path, tag string) ([]byte, error) {
	var conf loader.Config
	conf.Import(path)
	prog, err := conf.Load()
	if err != nil {
		return nil, err
	}
	pack := prog.Package(path)
	if pack == nil {
		return nil, errors.New("Package was not loaded.")
	}
	if pack.Types == nil {
		return nil, errors.New("Missing type information.")
	}

	var src bytes.Buffer
	for _, f := range pack.Files {
		var err error
		var lastIdent *ast.Ident
		ast.Inspect(f, func(n ast.Node) bool {
			switch n := n.(type) {
			case *ast.Ident:
				lastIdent = n
			case *ast.StructType:
				var def []byte
				def, err = generateStructDefinition(tag, pack, lastIdent, n)
				src.Write(def)
			}
			return true
		})
		if err != nil {
			return nil, err
		}
	}
	return format.Source(src.Bytes())
}

func generateStructDefinition(tagName string,
	info *loader.PackageInfo,
	ident *ast.Ident,
	node *ast.StructType,
) ([]byte, error) {
	var src bytes.Buffer
	fields := []graphQLFieldDefinition{}
	for _, field := range node.Fields.List {
		if field.Tag == nil {
			continue
		}
		tag := reflect.StructTag(strings.Trim(field.Tag.Value, "`"))
		name := tag.Get(tagName)
		if name == "" || name == "-" {
			continue
		}
		typ, ok := info.Types[field.Type]
		if !ok {
			continue
		}
		fields = append(fields, graphQLFieldDefinition{
			Name:        field.Names[0].Name,
			Type:        typ.Type,
			GraphQLName: name,
		})
	}
	if len(fields) == 0 {
		return nil, nil
	}
	packageName := info.Pkg.Name()
	err := generateGraphQLObjectDefinition(&src, graphQLObjectDefinition{
		Name:    ident.Name,
		Package: packageName,
		Fields:  fields,
	})
	if err != nil {
		return nil, err
	}
	return src.Bytes(), nil
}

const graphQLObjectTemplate = `
{{$obj := .}}
var {{.Name}}Type = graphql.NewObject(
	graphql.ObjectConfig{
		Name: "{{.Name}}",
		Fields: graphql.Fields{ {{range .Fields}}
			"{{.GraphQLName}}": &graphql.Field{
				Type:        {{.TypeString}},
				Description: "The {{.Name}} of the {{$obj.Name}}.", {{if .NeedsResolve}}
				Resolve: func(params graphql.ResolveParams) (interface{}, error) {
					if src := params.Source.(*{{$obj.Package}}.{{$obj.Name}}); src != nil {
						return *src.{{.Name}}, nil
					}
					return nil, errors.New("Invalid type")
				}, {{end}}
			}, {{end}}
		},
	},
)
`

type graphQLObjectDefinition struct {
	Name    string
	Package string
	Fields  []graphQLFieldDefinition
}

type graphQLFieldDefinition struct {
	Name        string
	Type        types.Type
	GraphQLName string
}

func (d graphQLFieldDefinition) NeedsResolve() bool {
	return strings.HasPrefix(d.Type.String(), "*")
}

func (d graphQLFieldDefinition) TypeString() string {
	typeMap := map[string]string{
		"string":  "graphql.String",
		"bool":    "graphql.Boolean",
		"int64":   "graphql.Int",
		"int32":   "graphql.Int",
		"int":     "graphql.Int",
		"float64": "graphql.Float",
	}

	typ := strings.ToLower(strings.TrimPrefix(d.Type.String(), "*"))
	mappedTyp, ok := typeMap[typ]
	if !ok {
		return "graphql.NewScalar(ScalarConfig{Name: \"" + d.Type.String() + "\"})"
	}
	if strings.HasPrefix(d.Type.String(), "*") {
		return fmt.Sprintf("graphql.NewNonNull(%s)", mappedTyp)
	}
	return mappedTyp
}

func generateGraphQLObjectDefinition(buf *bytes.Buffer,
	def graphQLObjectDefinition,
) error {
	templ, err := template.New("object").Parse(graphQLObjectTemplate)
	if err != nil {
		return err
	}
	return templ.Execute(buf, def)
}
