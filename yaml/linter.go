// Copyright © 2023 Meroxa, Inc.
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

package yaml

import (
	"fmt"

	"github.com/conduitio/evolviconf"
	"github.com/conduitio/yaml/v3"
)

type configLinter struct {
	// expandedChangelog contains a map of all changes in the changelog. The
	// first key is the version, the second is a map of changes in that version.
	// Changes are stored hierarchical in submaps. For example, if the field
	// x.y.z changed in version 1.2.3 the map will contain
	// { "1.2.3" : { "x" : { "y" : { "z" : Change{} } } } }.
	expandedChangelog map[string]map[string]any
}

func newConfigLinter(changelogs ...evolviconf.Changelog) *configLinter {
	// expand changelogs
	expandedChangelog := make(map[string]map[string]any)
	for _, changelog := range changelogs {
		for k, v := range changelog.Expand() {
			if _, ok := expandedChangelog[k]; ok {
				panic(fmt.Errorf("changelog already contains version %v", k))
			}
			expandedChangelog[k] = v
		}
	}

	return &configLinter{
		expandedChangelog: expandedChangelog,
	}
}

func (cl *configLinter) DecoderHook(version string, warn *evolviconf.Warnings) yaml.DecoderHook {
	return func(path []string, node *yaml.Node) {
		if w, ok := cl.InspectNode(version, path, node); ok {
			*warn = append(*warn, w)
		}
	}
}

func (cl *configLinter) InspectNode(version string, path []string, node *yaml.Node) (evolviconf.Warning, bool) {
	if c, ok := cl.findChange(version, path); ok {
		return cl.newWarning(path[len(path)-1], node, c.Message), true
	}
	return evolviconf.Warning{}, false
}

func (cl *configLinter) findChange(version string, path []string) (evolviconf.Change, bool) {
	curMap := cl.expandedChangelog[version]
	last := len(path) - 1
	for i, field := range path {
		nextMap, ok := curMap[field]
		if !ok {
			nextMap, ok = curMap["*"]
			if !ok {
				break
			}
		}
		switch v := nextMap.(type) {
		case map[string]any:
			curMap = v
			continue
		case evolviconf.Change:
			if i == last {
				return v, true
			}
		}
		break
	}
	return evolviconf.Change{}, false
}

func (cl *configLinter) newWarning(field string, node *yaml.Node, message string) evolviconf.Warning {
	return evolviconf.Warning{
		Position: evolviconf.Position{
			Field:  field,
			Line:   node.Line,
			Column: node.Column,
			Value:  node.Value,
		},
		Message: message,
	}
}
