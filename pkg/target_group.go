package gmakec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type TargetGroup struct {
	Targets []int
}

func (targetGroup *TargetGroup) Configure(defContext *DefinitionContext, defContexts *[]*DefinitionContext) ([]string, error) {
	buildCommands := []string{}

	for i := len(targetGroup.Targets) - 1; i >= 0; i-- {
		targetIndex := targetGroup.Targets[i]
		targetDef := defContext.Definition.Targets[targetIndex]

		targetDef.ExecuteHooks("preConfigure", defContext.DefinitionPath)

		// merge compiler flags
		compilerDef, err := targetDef.Compiler.WithRef(&defContext.Definition.Compilers)

		if err != nil {
			return nil, err
		}

		buildCommand := []string{
			fmt.Sprintf("%d", targetIndex),
			compilerDef.Path,
		}

		buildCommand = append(buildCommand, compilerDef.Flags...)

		for _, include := range targetDef.Includes {
			if strings.Contains(include, ":") {
				refStringArrayValue, err := FindRefTargetStringArrayValue(include, &targetDef, defContexts)

				if err != nil {
					return nil, err
				}

				for _, value := range refStringArrayValue {
					buildCommand = append(buildCommand, "-I")
					buildCommand = append(buildCommand, value)
				}
			} else {
				buildCommand = append(buildCommand, "-I")
				buildCommand = append(buildCommand, include)
			}
		}

		for _, link := range targetDef.Links {
			if len(link.Path) > 0 {
				buildCommand = append(buildCommand, "-L")
				linkPath := link.Path

				if strings.Contains(linkPath, ":") {
					linkPath, err = FindRefTargetStringValue(linkPath, &targetDef, defContexts)
				}

				buildCommand = append(buildCommand, filepath.Dir(linkPath))
			}

			buildCommand = append(buildCommand, link.Link)
		}

		buildCommand = append(buildCommand, "-o")
		buildCommand = append(buildCommand, targetDef.Output)

		if defContext.DefinitionPath == "." {
			if err := os.MkdirAll(filepath.Dir(targetDef.Output), os.ModePerm); err != nil {
				return nil, err
			}
		}

		for _, source := range targetDef.Sources {
			if strings.Contains(source, "*") {
				globbed, err := filepath.Glob(source)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, globbed...)
			} else if strings.Contains(source, ":") {
				refStringValue, err := FindRefTargetStringValue(source, &targetDef, defContexts)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, refStringValue)
			} else {
				buildCommand = append(buildCommand, source)
			}
		}

		buildCommands = append(buildCommands, strings.Join(buildCommand, " "))
		targetDef.ExecuteHooks("postConfigure", defContext.DefinitionPath)
	}

	return buildCommands, nil
}
