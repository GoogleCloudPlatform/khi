package testflags

import "flag"

var SkipCloudLogging = flag.Bool("skip-cloud-logging", false, "skip tests that require cloud logging")
