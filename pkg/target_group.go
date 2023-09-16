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

func (targetGroup *TargetGroup) Configure(globalDef *GlobalDefinition) ([]string, error) {
	buildCommands := []string{}

	for i := len(targetGroup.Targets) - 1; i >= 0; i-- {
		targetIndex := targetGroup.Targets[i]
		targetDef := globalDef.Targets[targetIndex]

		targetDef.ExecuteHooks("preConfigure")

		// merge compiler flags
		compilerDef, err := targetDef.Compiler.WithRef(&globalDef.Compilers)

		if err != nil {
			return nil, err
		}

		buildCommand := []string{
			fmt.Sprintf("%d", targetIndex),
			compilerDef.Path,
		}

		buildCommand = append(buildCommand, compilerDef.Flags...)

		for _, include := range targetDef.Includes {
			buildCommand = append(buildCommand, "-I")
			buildCommand = append(buildCommand, include)
		}

		for _, link := range targetDef.Links {
			if len(link.Path) > 0 {
				buildCommand = append(buildCommand, "-L")
				linkPath := link.Path

				if strings.Contains(linkPath, ":") {
					linkPath, err = globalDef.RefTargetStringValue(linkPath, &targetDef)
				}

				buildCommand = append(buildCommand, filepath.Dir(linkPath))
			}

			buildCommand = append(buildCommand, link.Link)
		}

		buildCommand = append(buildCommand, "-o")
		buildCommand = append(buildCommand, targetDef.Output)

		if err := os.MkdirAll(filepath.Dir(targetDef.Output), os.ModePerm); err != nil {
			return nil, err
		}

		for _, source := range targetDef.Sources {
			if strings.Contains(source, "*") {
				globbed, err := filepath.Glob(source)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, globbed...)
			} else if strings.Contains(source, ":") {
				refStringValue, err := globalDef.RefTargetStringValue(source, &targetDef)

				if err != nil {
					return nil, err
				}

				buildCommand = append(buildCommand, refStringValue)
			} else {
				buildCommand = append(buildCommand, source)
			}
		}

		buildCommands = append(buildCommands, strings.Join(buildCommand, " "))
		targetDef.ExecuteHooks("postConfigure")
	}

	return buildCommands, nil
}
