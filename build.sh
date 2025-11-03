#!/bin/bash -ex
cd "$(dirname "$0")"
go install tool
mage Build Coverage CrossCompile DemoWebserver
cat report.out
