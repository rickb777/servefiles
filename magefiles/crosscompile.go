// See https://magefile.org/

//go:build mage

// Build steps for the logrotate API:
package main

import (
	"log"
	"os"

	"github.com/magefile/mage/sh"
)

// tests the module on both amd64 and i386 architectures for Linux and Windows
func CrossCompile() error {
	win := "build"
	linux := "test"
	if os.Getenv("GOOS") == "windows" {
		win = "test"
		linux = "build"
	}
	log.Printf("Testing on Windows\n")
	if err := sh.RunWithV(map[string]string{"GOOS": "windows"}, "go", win, "./..."); err != nil {
		return err
	}
	for _, arch := range []string{"amd64", "386"} {
		log.Printf("Testing on Linux/%s\n", arch)
		env := map[string]string{"GOOS": "linux", "GOARCH": arch}
		if _, err := sh.Exec(env, os.Stdout, os.Stderr, "go", linux, "./..."); err != nil {
			return err
		}
	}
	return nil
}
