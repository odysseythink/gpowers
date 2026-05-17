//go:build windows

package main

import "syscall"

var sigTerm = syscall.Signal(15)
var sigKill = syscall.Signal(9)
