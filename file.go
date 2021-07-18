package main

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var (
	rComment = regexp.MustCompile(`^//\s*@inject_tag:\s*(.*)$`)
	rInject  = regexp.MustCompile("`.+`$")
	rTags    = regexp.MustCompile(`[\w_]+:"[^"]+"`)
	pbFile   = regexp.MustCompile(`\.pb.go`)
)

var withClean bool

type textArea struct {
	Start        int
	End          int
	CurrentTag   string
	InjectTag    string
	CommentStart int
	CommentEnd   int
}

func parseFile(inputPath string, xxxSkip []string) (areas []textArea, err error) {
	logf("parsing file %q for inject tag comments", inputPath)
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, inputPath, nil, parser.ParseComments)
	if err != nil {
		return
	}

	for _, decl := range f.Decls {
		// check if is generic declaration
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		var typeSpec *ast.TypeSpec
		for _, spec := range genDecl.Specs {
			if ts, tsOK := spec.(*ast.TypeSpec); tsOK {
				typeSpec = ts
				break
			}
		}

		// skip if can't get type spec
		if typeSpec == nil {
			continue
		}

		// not a struct, skip
		structDecl, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			continue
		}

		builder := strings.Builder{}
		if len(xxxSkip) > 0 {
			for i, skip := range xxxSkip {
				builder.WriteString(fmt.Sprintf("%s:\"-\"", skip))
				if i > 0 {
					builder.WriteString(",")
				}
			}
		}

		for _, field := range structDecl.Fields.List {
			// skip if field has no doc
			if len(field.Names) > 0 {
				name := field.Names[0].Name
				if len(xxxSkip) > 0 && strings.HasPrefix(name, "XXX") {
					currentTag := field.Tag.Value
					area := textArea{
						Start:      int(field.Pos()),
						End:        int(field.End()),
						CurrentTag: currentTag[1 : len(currentTag)-1],
						InjectTag:  builder.String(),
					}
					areas = append(areas, area)
				}
			}
			if field.Doc == nil {
				continue
			}
			for _, comment := range field.Doc.List {
				tag := tagFromComment(comment.Text)
				if tag == "" {
					continue
				}
				currentTag := field.Tag.Value
				area := textArea{
					Start:        int(field.Pos()),
					End:          int(field.End()),
					CurrentTag:   currentTag[1 : len(currentTag)-1],
					InjectTag:    tag,
					CommentStart: int(comment.Pos()),
					CommentEnd:   int(comment.End()),
				}
				areas = append(areas, area)
			}
		}
	}
	logf("parsed file %q, number of fields to inject custom tags: %d", inputPath, len(areas))
	return
}

func writeFile(inputPath string, areas []textArea) (err error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return
	}

	contents, err := ioutil.ReadAll(f)
	if err != nil {
		return
	}

	if err = f.Close(); err != nil {
		return
	}

	// inject custom tags from tail of file first to preserve order
	for i := range areas {
		area := areas[len(areas)-i-1]
		logf("inject custom tag %q to expression %q", area.InjectTag, string(contents[area.Start-1:area.End-1]))
		contents = injectTag(contents, area)
		if withClean {
			contents = cleanTag(contents, area)
		}
	}
	if err = ioutil.WriteFile(inputPath, contents, 0644); err != nil {
		return
	}

	if len(areas) > 0 {
		logf("file %q is injected with custom tags", inputPath)
	}
	return
}

func getFiles(dir string) []string {
	paths := []string{}
	files, err := os.ReadDir(dir)
	if err != nil {
		logf("read: error occurred reading files in dir: %s: %s", dir, err.Error())
		return paths
	}
	for _, file := range files {
		if err := fs.WalkDir(os.DirFS(dir), file.Name(), func(path string, d fs.DirEntry, err error) error {
			if pbFile.Match([]byte(path)) {
				paths = append(paths, filepath.Join(dir, path))
				return nil
			}
			return nil
		}); err != nil {
			logf("walkdir: error occurred reading files in dir: %s: %s", dir, err.Error())
			return paths
		}
	}
	return paths
}

func process(inputFile string, xxxSkipSlice []string) error {
	logf("processing file: %s\n", inputFile)
	areas, err := parseFile(inputFile, xxxSkipSlice)
	if err != nil {
		return err
	}
	if err = writeFile(inputFile, areas); err != nil {
		return err
	}
	return nil
}
