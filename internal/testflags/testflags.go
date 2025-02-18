package testflags

import "flag"

// SkipCloudLogging is a flag of program arguments on testing.  Unit tests using Cloud Logging APIs are skipped with this flag.
var SkipCloudLogging = flag.Bool("skip-cloud-logging", false, "skip tests that require cloud logging")
