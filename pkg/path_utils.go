package gmakec

import (
	"log"
	"os"
	"path/filepath"
)

func RemovePath(path string) {
	if err := os.RemoveAll(path); err != nil {
		log.Printf("WARNING: could not remove path %s: %s\n", path, err.Error())
	}
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
