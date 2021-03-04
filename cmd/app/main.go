package main

import "github.com/fancar/wrenches/cmd/app/cmd"

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
