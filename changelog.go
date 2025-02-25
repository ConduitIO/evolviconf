// Copyright © 2024 Meroxa, Inc.
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

package evolviconf

import (
	"maps"
	"sort"
	"strings"

	"github.com/Masterminds/semver/v3"
)

// Changelog contains a map of all changes introduced in each version. Based on
// the changelog the parser can output warnings.
type Changelog map[*semver.Version][]Change

// Change is a single change that was introduced in a specific version.
type Change struct {
	// field contains the path to the field that was changed. Nested fields can
	// be represented with dots (e.g. nested.field)
	Field      string
	ChangeType ChangeType
	// message is the log message that will be printed if a file is detected
	// that uses this field with an unsupported version.
	Message string
}

// ChangeType defines the type of the change introduced in a specific version.
type ChangeType int

const (
	FieldDeprecated ChangeType = iota
	FieldIntroduced
)

// Expand expands a changelog map into a structure that is useful for traversing
// in ConfigLinter. It returns a map of all versions and their changes. The
// changes are stored in a nested map where each token in the field is a key in
// the map. This allows for easy traversal of the changes in a linter.
func (cl Changelog) Expand() map[*semver.Version]map[string]any {
	var versions semver.Collection
	for k := range maps.Keys(cl) {
		versions = append(versions, k)
	}
	sort.Sort(versions)

	knownChanges := make(map[*semver.Version]map[string]any)
	for _, v := range versions {
		knownChanges[v] = make(map[string]any)
	}

	for _, v := range versions {
		changes := cl[v]
		for _, v2 := range versions {
			switch {
			case !v.GreaterThan(v2):
				// warn about deprecated fields in future versions
				for _, c := range changes {
					if c.ChangeType == FieldDeprecated {
						cl.addChange(c, knownChanges[v2])
					}
				}
			case v.GreaterThan(v2):
				// warn about introduced fields in older versions
				for _, c := range changes {
					if c.ChangeType == FieldIntroduced {
						cl.addChange(c, knownChanges[v2])
					}
				}
			}
		}
	}
	return knownChanges
}

// addChange stores change c in map m by splitting c.field into multiple
// tokens and storing it in m[token1][token2][...][tokenN]. If any map in
// that hierarchy does not exist it is created. If any value in that
// hierarchy exists and is _not_ a map it is _not_ replaced. This means that
// changes related to parent fields take precedence over changes related to
// child fields.
func (cl Changelog) addChange(change Change, m map[string]any) {
	tokens := strings.Split(change.Field, ".")
	curMap := m
	for i, t := range tokens {
		if i == len(tokens)-1 {
			// last token, set it in the map
			curMap[t] = change
			break
		}
		raw, ok := curMap[t]
		if !ok {
			nextMap := make(map[string]any)
			curMap[t] = nextMap
			curMap = nextMap
			continue
		}
		curMap, ok = raw.(map[string]any)
		if !ok {
			break
		}
	}
}
