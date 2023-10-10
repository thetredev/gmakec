#!/bin/bash

set -e

ROOT_DIR=$(git rev-parse --show-toplevel)
PROJECT_BINARY=${ROOT_DIR}/build/gmakec

go get
go build -o ${PROJECT_BINARY}

cd ${ROOT_DIR}/examples

echo "EXAMPLE DIRS"
example_dirs=$(realpath $(find . -name .gmakec))

for example_dir in ${example_dirs}; do
    cd $(dirname ${example_dir})

    echo "RELATIVE DIR"
    relative_path=$(realpath --relative-to=${ROOT_DIR} $(pwd))
    echo "Building: ${relative_path}"

    ${PROJECT_BINARY} rebuild

    for example_binary in $(ls -1 -d build/*); do
        example_binary_path="${relative_path}/${example_binary}"

        if [[ ${example_binary} == *.o ]]; then
            echo "${example_binary_path} is object file, skipping..."
            continue
        fi

        if [[ ${example_binary} == *.so ]]; then
            echo "${example_binary_path} is shared library file, skipping..."
            continue
        fi

        if [[ ${example_binary} == *.a ]]; then
            echo "${example_binary_path} is static library file, skipping..."
            continue
        fi

        echo "Running: ${relative_path}/${example_binary}"

        if [ -f ./run.sh ]; then
            ./run.sh
        else
            ./${example_binary}
        fi
    done
    echo
done

cd ${ROOT_DIR}
