#!/bin/bash -e

projectname=dmgo

pkg=github.com/theinternetftw/$projectname/cmd/$projectname

if [ "$1" == "f" ]
then
    set -x
    go vet
    golint
    go build $pkg
    exit
fi

echo 
echo "$projectname buildscript"
echo "possible args: f, all, profiling.{cpu|mem|block|live}, release"
echo

echo "running fmt, vet, etc..."
echo

# go fmt
goimports -w *.go cmd/*/*.go
go vet
go vet ./cmd/*

if [ "$1" == "" ]
then
    set -x
    go build $pkg

elif [ "$1" == "all" ]
then
    set -x
    env GOOS=windows GOARCH=amd64 go build -o build-dev/$projectname-win-x64.exe $pkg
    env GOOS=windows GOARCH=386 go build -o build-dev/$projectname-win-x86.exe $pkg

    env GOOS=linux GOARCH=amd64 go build -o build-dev/$projectname-linux-x64 $pkg
    env GOOS=linux GOARCH=386 go build -o build-dev/$projectname-linux-x86 $pkg

    env GOOS=darwin GOARCH=amd64 go build -o build-dev/$projectname-mac-x64 $pkg
    env GOOS=darwin GOARCH=386 go build -o build-dev/$projectname-mac-x86 $pkg

    env GOOS=linux GOARCH=arm GOARM=6 go build -o build-dev/$projectname-rpi $pkg
    env GOOS=linux GOARCH=arm GOARM=7 go build -o build-dev/$projectname-rpi2 $pkg

elif [ "$1" == "release" ]
then
    set -x
    env GOOS=windows GOARCH=amd64 go build -tags release -o build/$projectname-win-x64.exe $pkg
    env GOOS=windows GOARCH=386 go build -tags release -o build/$projectname-win-x86.exe $pkg

    env GOOS=linux GOARCH=amd64 go build -tags release -o build/$projectname-linux-x64 $pkg
    env GOOS=linux GOARCH=386 go build -tags release -o build/$projectname-linux-x86 $pkg

    env GOOS=darwin GOARCH=amd64 go build -tags release -o build/$projectname-mac-x64 $pkg
    env GOOS=darwin GOARCH=386 go build -tags release -o build/$projectname-mac-x86 $pkg

    env GOOS=linux GOARCH=arm GOARM=6 go build -tags release -o build/$projectname-rpi $pkg
    env GOOS=linux GOARCH=arm GOARM=7 go build -tags release -o build/$projectname-rpi2 $pkg

elif [ "$1" == "profiling" ]; then
    echo not like that, like this: profiling.cpu, profiling.mem, etc.

elif [ "$1" == "profiling.cpu" ]
then
    set -x
    go build -tags profiling_cpu $pkg

elif [ "$1" == "profiling.mem" ]
then
    set -x
    go build -tags profiling_mem $pkg

elif [ "$1" == "profiling.block" ]
then
    set -x
    go build -tags profiling_block $pkg

elif [ "$1" == "profiling.live" ]
then
    set -x
    go build -tags profiling_live $pkg

else
    echo unknown arg

fi
