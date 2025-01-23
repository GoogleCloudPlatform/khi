// Copyright 2024 Google LLC
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package flag

import (
	"context"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
)

// This package provides helper functions to parse command line arguments and environment variables.
// It provides functions similar to the standard `flag` package, but with added functionality to handle environment variables as a fallback.  The package also provides a mechanism to dump all parsed flag values.

type FlagParser func() error
type FlagValueDumper func() string

var flagParsed bool = false
var flagParsers []FlagParser = []FlagParser{}
var flagValueDumper []FlagValueDumper = []FlagValueDumper{}

var envFalsyValues map[string]struct{} = map[string]struct{}{
	"false": {},
}

func Parse() error {
	flag.Parse()
	for _, parser := range flagParsers {
		err := parser()
		if err != nil {
			return err
		}
	}
	flagParsed = true
	return nil
}

func Parsed() bool {
	return flagParsed
}

func Reset() {
	flagParsed = false
	flagParsers = []FlagParser{}
	flagValueDumper = []FlagValueDumper{}
}

func DumpAll(ctx context.Context) {
	setting := ""
	for _, dumper := range flagValueDumper {
		setting += dumper() + "\n"
	}
	slog.InfoContext(ctx, setting)
}

func PrintDefaults() {
	flag.PrintDefaults()
}

// String is a similar to flag.String but it also parse value from specified environment variable when the parameter was not provided explicitly on command line argument.
// Environment variables are ignored when the given envKey is an empty string.
func String(name string, value string, usage string, envKey string) *string {
	result := value
	resultPtr := &result
	if envKey != "" {
		usage = fmt.Sprintf("%s [environment variable key: \"%s\"]", usage, envKey)
	}
	fromCmdArgs := flag.String(name, value, usage)
	flagParsers = append(flagParsers, func() error {
		providedFromCmdArgs, err := isProvidedFromCommandlineArgs(name)
		if err != nil {
			return err
		}
		if providedFromCmdArgs {
			*resultPtr = *fromCmdArgs
		} else if isProvidedFromEnvironmentVariable(envKey) {
			*resultPtr = os.Getenv(envKey)
		}
		return nil
	})
	flagValueDumper = append(flagValueDumper, func() string {
		return fmt.Sprintf("%s: %v", name, *resultPtr)
	})
	return resultPtr
}

// Bool is a similar to flag.Bool but it also parse value from specified environment variable when the parameter was not provided explicitly on command line argument.
// Environment variables are ignored when the given envKey is an empty string.
func Bool(name string, value bool, usage string, envKey string) *bool {
	result := value
	resultPtr := &result
	if envKey != "" {
		usage = fmt.Sprintf("%s [environment variable key: \"%s\"]", usage, envKey)
	}
	fromCmdArgs := flag.Bool(name, value, usage)
	flagParsers = append(flagParsers, func() error {
		providedFromCmdArgs, err := isProvidedFromCommandlineArgs(name)
		if err != nil {
			return err
		}
		if providedFromCmdArgs {
			*resultPtr = *fromCmdArgs
		} else if isProvidedFromEnvironmentVariable(envKey) {
			value := os.Getenv(envKey)
			if _, found := envFalsyValues[strings.ToLower(value)]; found {
				*resultPtr = false
			} else {
				*resultPtr = true
			}
		}
		return nil
	})
	flagValueDumper = append(flagValueDumper, func() string {
		return fmt.Sprintf("%s: %v", name, *resultPtr)
	})
	return resultPtr
}

// Int is a similar to flag.Int but it also parse value from specified environment variable when the parameter was not provided explicitly on command line argument.
// Environment variables are ignored when the given envKey is an empty string.
func Int(name string, value int, usage string, envKey string) *int {
	result := value
	resultPtr := &result
	if envKey != "" {
		usage = fmt.Sprintf("%s [environment variable key: \"%s\"]", usage, envKey)
	}
	fromCmdArgs := flag.Int(name, value, usage)
	flagParsers = append(flagParsers, func() error {
		providedFromCmdArgs, err := isProvidedFromCommandlineArgs(name)
		if err != nil {
			return err
		}
		if providedFromCmdArgs {
			*resultPtr = *fromCmdArgs
		} else if isProvidedFromEnvironmentVariable(envKey) {
			envValue := os.Getenv(envKey)
			intEnv, err := strconv.ParseInt(envValue, 10, 64)
			if err != nil {
				return err
			}
			*resultPtr = int(intEnv)
		}
		return nil
	})
	flagValueDumper = append(flagValueDumper, func() string {
		return fmt.Sprintf("%s: %v", name, *resultPtr)
	})
	return resultPtr
}

func isProvidedFromCommandlineArgs(key string) (bool, error) {
	if !flag.Parsed() {
		return false, fmt.Errorf("command line arguments are not yet parsed")
	}
	found := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == key {
			found = true
		}
	})
	return found, nil
}

func isProvidedFromEnvironmentVariable(key string) bool {
	if key == "" {
		return false
	}
	_, found := os.LookupEnv(key)
	return found
}
