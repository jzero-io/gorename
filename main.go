package main

import (
	"bytes"
	"fmt"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/fatih/color"
	"github.com/urfave/cli"
	"golang.org/x/tools/go/ast/astutil"
)

func main() {
	app := cli.NewApp()
	app.Name = "gorename"
	app.Usage = "Rename golang package"
	app.Version = "v1.3.0"
	app.ArgsUsage = "[source file or directory path] [old package name] [new package name]"
	app.Author = "jzero-io"

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "source, s",
			Value: "./",
			Usage: "source package path or file path",
		},
		cli.BoolTFlag{
			Name:  "quiet, q",
			Usage: "do not print",
		},
	}

	app.Action = func(c *cli.Context) {
		source := c.String("source")
		from := c.Args().Get(0)
		to := c.Args().Get(1)
		quiet := c.Bool("quiet")

		fileInfo, err := os.Stat(source)
		if err != nil {
			cli.HandleExitCoder(cli.NewExitError("source is not a directory or file", -1))
			return
		}

		if from == "" || to == "" {
			if err = cli.ShowAppHelp(c); err != nil {
				cli.HandleExitCoder(cli.NewExitError(err, -1))
			}
			return
		}

		if !quiet {
			fmt.Println(color.GreenString("[INFO]"), "start update import ", from, " to ", to)
		}

		// rename dir name
		if _, err := os.Stat(filepath.Base(source)); err == nil {
			if err = os.Rename(filepath.Base(from), filepath.Base(to)); err != nil {
				// cli.HandleExitCoder(cli.NewExitError(err, -1))
				return
			}
		}

		if !fileInfo.IsDir() {
			err = ProcessFile(source, from, to, quiet)
		} else {
			err = ProcessDir(source, from, to, quiet)
		}
		if err != nil {
			fmt.Println(color.YellowString("[WARN]"), err.Error())
		} else {
			if !quiet {
				fmt.Println(color.GreenString("[INFO] success!"))
			}
		}
	}
	if err := app.Run(os.Args); err != nil {
		cli.HandleExitCoder(cli.NewExitError(err, -1))
	}
}

func ProcessDir(dir string, from string, to string, quiet bool) error {
	return filepath.Walk(dir, func(filepath string, info os.FileInfo, err error) error {
		if path.Ext(filepath) == ".go" {
			if err = ProcessFile(filepath, from, to, quiet); err != nil {
				return err
			}
		}
		return nil
	})
}

func ProcessFile(filePath string, from string, to string, quiet bool) error {
	fSet := token.NewFileSet()

	file, err := parser.ParseFile(fSet, filePath, nil, parser.ParseComments)

	if err != nil {
		fmt.Println(err)
	}

	imports := astutil.Imports(fSet, file)

	changeNum := 0

	for _, tempPackage := range imports {
		for _, mImport := range tempPackage {
			importString := strings.TrimSuffix(strings.TrimPrefix(mImport.Path.Value, "\""), "\"")

			if strings.HasPrefix(importString, from) {
				changeNum++

				replacePackage := strings.ReplaceAll(importString, from, to)

				if mImport.Name != nil && len(mImport.Name.Name) > 0 {
					astutil.DeleteNamedImport(fSet, file, mImport.Name.Name, importString)
					astutil.AddNamedImport(fSet, file, mImport.Name.Name, replacePackage)
				} else {
					astutil.DeleteImport(fSet, file, importString)
					astutil.AddImport(fSet, file, replacePackage)
				}
			}
		}
	}

	if changeNum > 0 {
		var outputBuffer bytes.Buffer

		err = printer.Fprint(&outputBuffer, fSet, file)
		if err != nil {
			return err
		}

		if err = os.WriteFile(filePath, outputBuffer.Bytes(), os.ModePerm); err != nil {
			return err
		}

		change := "change"

		if changeNum > 1 {
			change += "s"
		}

		if !quiet {
			fmt.Println(color.GreenString("[INFO]"), changeNum, change, "in file", filePath)
		}
	}

	return nil
}
