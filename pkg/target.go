package gmakec

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/exp/slices"
)

type Target struct {
	Definition *TargetDefinition
	Index      int
	Defines    []string
	Includes   []string
	Links      []string
	Sources    []string
}

func collectModTimes(path string) ([]int64, error) {
	modTimes := []int64{}

	err := filepath.Walk(path, func(name string, info os.FileInfo, err error) error {
		if name == path && info != nil && info.IsDir() {
			return nil
		}

		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}

			return err
		}

		modTimes = append(modTimes, info.ModTime().UnixMicro())
		return nil
	})

	if err != nil {
		return nil, err
	}

	return modTimes, nil
}

func collectModTimesMultiple(paths []string) ([]int64, error) {
	collected := []int64{}

	for _, path := range paths {
		modTimes, err := collectModTimes(path)

		if err != nil {
			return nil, err
		}

		collected = append(collected, modTimes...)
	}

	return collected, nil
}

func (this *Target) needsRebuild() (bool, error) {
	outputModTimes, err := collectModTimes(this.Definition.Output)

	if err != nil {
		return false, err
	}

	if len(outputModTimes) == 0 {
		return true, nil
	}

	outputMax := slices.Max(outputModTimes)
	includeModTimes, err := collectModTimesMultiple(this.Includes)

	if err != nil {
		return false, err
	}

	linkModTimes, err := collectModTimesMultiple(this.Links)

	if err != nil {
		return false, err
	}

	sourceModTimes, err := collectModTimesMultiple(this.Sources)

	if err != nil {
		return false, err
	}

	if len(includeModTimes) > 0 && slices.Max(includeModTimes) > outputMax {
		return true, nil
	}

	if len(linkModTimes) > 0 && slices.Max(linkModTimes) > outputMax {
		return true, nil
	}

	if len(sourceModTimes) > 0 && slices.Max(sourceModTimes) > outputMax {
		return true, nil
	}

	return false, nil
}

func (this *Target) buildCommand() (string, error) {
	rebuild, err := this.needsRebuild()

	if err != nil {
		return "", err
	}

	command := []string{
		fmt.Sprintf("%d", this.Index),
	}

	if rebuild {
		command = append(command, "build")
	} else {
		command = append(command, "skip")
	}

	command = append(command, this.Definition.Compiler.Object.Path)

	command = append(command, this.Definition.Compiler.Flags...)
	command = append(command, this.Defines...)
	command = append(command, this.Includes...)
	command = append(command, this.Links...)

	command = append(command, this.Definition.Compiler.Object.OutputFlag)
	command = append(command, this.Definition.Output)

	command = append(command, this.Sources...)
	return strings.Join(command, " "), nil
}
