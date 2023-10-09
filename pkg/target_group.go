package gmakec

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/yargevad/filepathx"
)

type TargetGroup struct {
	Targets []int
}

func (this *TargetGroup) configure(
	definitionContext *DefinitionContext, definitionContexts *[]*DefinitionContext,
) ([]Target, error) {
	targets := []Target{}

	for i := len(this.Targets) - 1; i >= 0; i-- {
		targetIndex := this.Targets[i]
		targetDef := definitionContext.Definition.Targets[targetIndex]

		if err := targetDef.mergeHookRefs(targetIndex, definitionContext); err != nil {
			return nil, err
		}

		if err := targetDef.executeHooks("preConfigure", definitionContext.DefinitionPath); err != nil {
			return nil, err
		}

		compilerDef, err := targetDef.Compiler.sanitize(&definitionContext.Definition.Compilers)

		if err != nil {
			return nil, err
		}

		for _, configureFile := range targetDef.ConfigureFiles {
			if err := configureFile.Execute(definitionContext); err != nil {
				return nil, err
			}
		}

		targetDefCopy := targetDef
		targetDefCopy.Compiler = *compilerDef

		target := Target{
			Definition: &targetDefCopy,
			Index:      targetIndex,
		}

		for _, define := range targetDef.Defines {
			target.Defines = append(target.Defines, compilerDef.Object.DefineFlag)
			target.Defines = append(target.Defines, define)
		}

		for _, include := range targetDef.Includes {
			includeStrings := []string{}

			if strings.Contains(include, ":") {
				refStringArrayValue, err := findRefTargetStringArrayValue(include, &targetDef, definitionContexts)

				if err != nil {
					return nil, err
				}

				includeStrings = append(includeStrings, refStringArrayValue...)
			} else {
				includeStrings = append(includeStrings, include)
			}

			for _, includeString := range includeStrings {
				if strings.Contains(includeString, "*") {
					globbed, err := filepathx.Glob(includeString)

					if err != nil {
						return nil, err
					}

					for _, match := range globbed {
						f, _ := os.Stat(match)

						if f.IsDir() {
							target.Includes = append(target.Includes, compilerDef.Object.IncludeSearchFlag)
							target.Includes = append(target.Includes, match)
						}
					}
				} else {
					target.Includes = append(target.Includes, compilerDef.Object.IncludeSearchFlag)
					target.Includes = append(target.Includes, includeString)
				}
			}
		}

		for _, link := range targetDef.Links {
			if len(link.Path) > 0 {
				target.Links = append(target.Links, compilerDef.Object.LinkSearchFlag)
				linkPath := link.Path

				if strings.Contains(linkPath, ":") {
					linkPath, err = findRefTargetStringValue(linkPath, &targetDef, definitionContexts)
				}

				target.Links = append(target.Links, filepath.Dir(linkPath))
			}

			target.Links = append(target.Links, link.Link)
		}

		for _, source := range targetDef.Sources {
			if len(source.Platform) > 0 && runtime.GOOS != source.Platform {
				continue
			}

			if strings.Contains(source.Path, "*") {
				globbed, err := filepathx.Glob(source.Path)

				if err != nil {
					return nil, err
				}

				target.Sources = append(target.Sources, globbed...)
			} else if strings.Contains(source.Path, ":") {
				refStringValue, err := findRefTargetStringValue(source.Path, &targetDef, definitionContexts)

				if err != nil {
					return nil, err
				}

				target.Sources = append(target.Sources, refStringValue)
			} else {
				target.Sources = append(target.Sources, source.Path)
			}
		}

		targets = append(targets, target)

		if err = targetDef.executeHooks("postConfigure", definitionContext.DefinitionPath); err != nil {
			return nil, err
		}
	}

	return targets, nil
}
