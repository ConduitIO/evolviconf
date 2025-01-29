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

package v1

import (
	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/evolviconf/examples/app"
)

// Changelog contains a list of changes to the configuration file.
// The parser will output warnings based on the changelog.
var Changelog = evolviconf.Changelog{
	semver.MustParse("1.0"): {}, // initial version
	semver.MustParse("1.1"): {
		{
			Field:      "authToken",
			ChangeType: evolviconf.FieldIntroduced,
			Message:    "authToken is a field introduced in 1.1",
		},
	},
	semver.MustParse("1.2"): {
		{
			Field:      "port",
			ChangeType: evolviconf.FieldDeprecated,
			Message:    "port is deprecated in 1.2, and will be removed in a future version",
		},
	},
}

// YAMLConfiguration is the struct that corresponds to a version of a YAML configuration.
type YAMLConfiguration struct {
	Version   string `yaml:"version"`
	Host      string `yaml:"host"`
	Port      string `yaml:"port"`
	AuthToken string `yaml:"authToken"`
}

// ToConfig needs to be implemented so that EvolviConf can convert the YAML representation
// into app.Configuration.
func (s YAMLConfiguration) ToConfig() (app.Configuration, error) {
	return app.Configuration{
		Host: s.Host,
		Port: s.Port,
	}, nil
}
