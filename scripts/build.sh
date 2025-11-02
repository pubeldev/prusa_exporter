#!/bin/bash

set -eou pipefail

platforms=("linux/amd64" "linux/arm" "linux/riscv64" "linux/arm64" "windows/amd64" "darwin/amd64" "windows/arm64" "darwin/arm64")

cd $(dirname $0)
output_dir="../bin"

mkdir -p $output_dir

for platform in "${platforms[@]}"
do
    GOOS=$(echo $platform | cut -d'/' -f1)
    GOARCH=$(echo $platform | cut -d'/' -f2)

    output_name="$output_dir/prusa_exporter-$GOOS-$GOARCH"
    if [ $GOOS = "windows" ]; then
        output_name="$output_name.exe"
    fi

    echo "Building $output_name ..."    
    env GOOS=$GOOS GOARCH=$GOARCH go build -o $output_name ../.
done

echo "Build completed."
