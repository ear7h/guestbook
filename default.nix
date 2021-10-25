{ pkgs ? import <nixpkgs> {} }:
pkgs.runCommandLocal "guestbook" { src = ./.; nativeBuildInputs = [ pkgs.go ]; } ''
  cd $src

  export GOCACHE=$TMPDIR/gocache
  export GOPATH=$TMPDIR/go

  go build -o $out/bin/guestbook .
''
