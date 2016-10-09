#!/bin/bash


GITCOMMIT=$(git describe --match 'v[0-9]*' --dirty='.m' --always)
BUILDTIME=$(date -u '+%Y%m%d.%I%M%S%p')
version=0.0.2

echo "creating shadowsocks binary version $version"

ROOT=`pwd`
bindir=$ROOT/bin
mkdir -p $bindir

build() {
    local name
    local GOOS
    local GOARCH

    if [[ $1 == "darwin" ]]; then
        # Enable CGO for OS X so change network location will not cause problem.
        export CGO_ENABLED=1
    else
        export CGO_ENABLED=0
    fi

    prog=shadowsocks-$4
    pushd main/$4
    name=$prog-$3-$version
    echo "building $name"
    GOOS=$1 GOARCH=$2 go build -a -installsuffix cgo -ldflags "-X `go list shadowsocks/version`.VERSION=$version -X `go list shadowsocks/version`.BUILDTIME=$BUILDTIME -X `go list shadowsocks/version`.GITCOMMIT=$GITCOMMIT -w" -o $prog || exit 1
    if [[ $1 == "windows" ]]; then
        mv $prog $prog.exe
        zip $name.zip ../../conf/sample-$4.json $prog.exe
        mv $name.zip $bindir
        rm $prog.exe
    else
        # gzip -f $prog
        cp ../../conf/sample-$4.json sample-$4.json 
        tar -vzcf $name.tar.gz $prog sample-$4.json
        # gzip -9 -c $prog > $name.gz
        mv $name.tar.gz $bindir
        rm $prog sample-$4.json 
    fi
    popd
}

build darwin amd64 mac64 client
build linux amd64 linux64 client
build linux 386 linux32 client
build windows amd64 win64 client
build windows 386 win32 client

build darwin amd64 mac64 server
build linux amd64 linux64 server
build linux 386 linux32 server
build windows amd64 win64 server
build windows 386 win32 server

#script/createdeb.sh amd64
#script/createdeb.sh 386
#mv shadowsocks-go_$version-1-*.deb bin/
#rm -rf shadowsocks-go_$version-1*
