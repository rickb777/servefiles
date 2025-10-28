// See https://magefile.org/

//go:build mage

// Build steps for the servefiles API:
package main

import (
	"github.com/magefile/mage/sh"
)

var Default = DemoWebserver

func Build() error {
	if err := sh.RunV("go", "test", "./..."); err != nil {
		return err
	}
	if err := sh.RunV("gofmt", "-l", "-w", "-s", "."); err != nil {
		return err
	}
	if err := sh.RunV("go", "vet", "./..."); err != nil {
		return err
	}
	return nil
}

func DemoWebserver() error {
	if err := Build(); err != nil {
		return err
	}
	return sh.RunV("go", "build", "-o", "simple-webserver", "./webserver")
}
