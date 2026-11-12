// Copyright 2026 Google LLC
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

package astcheck

import (
	"fmt"
	"path"
	"strings"
)

var bannedExactStems = map[string]struct{}{
	"util":    {},
	"utils":   {},
	"helper":  {},
	"helpers": {},
	"common":  {},
	"misc":    {},
	"shared":  {},
}

var bannedStemSuffixes = []string{
	"_util",
	"_utils",
	"_helper",
	"_helpers",
	"_misc",
}

// CheckCatchAllFilename reports whether relPath uses a banned low-cohesion catch-all filename.
func CheckCatchAllFilename(relPath string) (string, bool) {
	base := path.Base(strings.TrimSpace(relPath))
	if !strings.HasSuffix(base, ".go") {
		return "", false
	}
	stem := strings.TrimSuffix(strings.TrimSuffix(base, ".go"), "_test")
	if _, bad := bannedExactStems[stem]; bad {
		return fmt.Sprintf("file %q uses catch-all name %q; name files after their domain responsibility", base, stem+".go"), true
	}
	for _, suffix := range bannedStemSuffixes {
		if strings.HasSuffix(stem, suffix) {
			return fmt.Sprintf("file %q uses catch-all suffix %q; group code into domain-named files", base, suffix+".go"), true
		}
	}
	return "", false
}
