package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v2"

	gmakec "github.com/thetredev/gmakec/pkg"
)

const GLOBAL_DEFINITION_YAML string = "gmakec.yaml"

var definitionContexts []*gmakec.DefinitionContext

func collectDefinitionContexts(defContext *gmakec.DefinitionContext) error {
	for _, defImport := range defContext.Definition.Imports {
		importedDefContext, err := gmakec.NewDefinitionContext(filepath.Join(defImport, GLOBAL_DEFINITION_YAML))

		if err != nil {
			return err
		}

		err = collectDefinitionContexts(importedDefContext)

		if err != nil {
			return err
		}
	}

	definitionContexts = append(definitionContexts, defContext)
	return nil
}

func configure(context *cli.Context) error {
	targets := context.Args().Slice()

	definitionContexts = make([]*gmakec.DefinitionContext, 0)
	defContext, err := gmakec.NewDefinitionContext(GLOBAL_DEFINITION_YAML)

	if err != nil {
		return err
	}

	err = collectDefinitionContexts(defContext)

	if err != nil {
		return nil
	}

	for _, dc := range definitionContexts {
		err = dc.Configure(&definitionContexts, targets...)

		if err != nil {
			return err
		}
	}

	return nil
}

func build(context *cli.Context) error {
	err := configure(context)

	if err != nil {
		return err
	}

	verbose := context.Args().Get(0) == "verbose"

	for _, dc := range definitionContexts {
		err = dc.Build(verbose)

		if err != nil {
			return err
		}
	}

	return nil
}

func clean(context *cli.Context) error {
	defContext, err := gmakec.NewDefinitionContext(GLOBAL_DEFINITION_YAML)

	if err != nil {
		return err
	}

	err = collectDefinitionContexts(defContext)

	if err != nil {
		return nil
	}

	for _, dc := range definitionContexts {
		for _, targetDef := range dc.Definition.Targets {
			outputDir := filepath.Join(dc.DefinitionPath, filepath.Dir(targetDef.Output))
			gmakec.RemovePath(outputDir)
		}

		gmakec.RemovePath(dc.ConfigureDir)
	}

	return nil
}

func rebuild(context *cli.Context) error {
	if err := clean(context); err != nil {
		return err
	}

	if err := build(context); err != nil {
		return err
	}

	return nil
}

func reconfigure(context *cli.Context) error {
	if err := clean(context); err != nil {
		return err
	}

	if err := configure(context); err != nil {
		return err
	}

	return nil
}

func main() {
	app := &cli.App{
		DefaultCommand: "build",
		Commands: []*cli.Command{
			{
				Name:   "configure",
				Usage:  "configure the project",
				Action: configure,
			},
			{
				Name:   "build",
				Usage:  "build the project",
				Action: build,
			},
			{
				Name:   "clean",
				Usage:  "rm -rf the output files",
				Action: clean,
			},
			{
				Name:   "reconfigure",
				Usage:  "Shorthand for clean + configure",
				Action: reconfigure,
			},
			{
				Name:   "rebuild",
				Usage:  "Shorthand for clean + build",
				Action: rebuild,
			},
		},
	}

	gmakec.InitCompilers()

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
