#!/bin/bash -e

projectname=dmgo
cmd_pkg_dir=./cmd/$projectname

echo "$projectname buildscript"
echo
echo "  optional args:"
echo "    all - build for all platforms."
echo "    release - build in the build_release folder (otherwise builds in build_dev) and"
echo "              adds the \"release\" build tag."
echo "    any other arg - inserted as a build tag."
echo
echo "  useful tags: release, profiling_cpu, profiling_mem, profiling_block, profiling_live"
echo

echo "running fmt, vet, etc..."
echo
goimports -w *.go cmd/*/*.go
go vet . ./cmd/*

build_folder="build_dev"
while [ "$#" -ne 0 ]; do
    case "$1" in
        "all") build_all_platforms=1 ;;
        "release") build_folder="build_release" ;& # fallthrough
        *) build_tags="$build_tags$1," ;;
    esac
    shift
done

mkdir -p "$build_folder"
if [ $build_tags ]; then
    build_tags="-tags $build_tags"
fi
if [ $build_all_platforms ]; then
    set -x
    env GOOS=windows GOARCH=amd64 go build $build_tags -o $build_folder/$projectname-win-x64.exe $cmd_pkg_dir
    env GOOS=linux GOARCH=amd64 go build $build_tags -o $build_folder/$projectname-linux-x64 $cmd_pkg_dir
    env GOOS=darwin GOARCH=amd64 go build $build_tags -o $build_folder/$projectname-mac-x64 $cmd_pkg_dir
    env GOOS=linux GOARCH=arm GOARM=6 go build $build_tags -o $build_folder/$projectname-rpi $cmd_pkg_dir
    env GOOS=linux GOARCH=arm GOARM=7 go build $build_tags -o $build_folder/$projectname-rpi2 $cmd_pkg_dir
else
    set -x
    cd $build_folder # to avoid -o option, which requires manually setting a different file extension on windows
    go build $build_tags ../$cmd_pkg_dir
fi

