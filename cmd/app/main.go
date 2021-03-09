package main

import "github.com/fancar/wrenches/cmd/app/cmd"

/*
	+++++++++++++++++ wrenches by наитие +++++++++++++++++++++
	All these are about additional tools for iot-network server
	govnocoded by Alexander Mamaev aka FancaR during deep trip
	fancarster@gmail.com | tg: @fancar
	+++++++++++++++++++++++ 2020 +++++++++++++++++++++++++++++
*/

var version string // set by the compiler

func main() {
	cmd.Execute(version)
}
