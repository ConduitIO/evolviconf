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

package examples

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/evolviconf/evolviyaml"
	"github.com/conduitio/evolviconf/examples/app"
	v1 "github.com/conduitio/evolviconf/examples/v1"
	"github.com/conduitio/yaml/v3"
)

func ExampleParseOlderConfigWithNewField() {
	constraint, err := semver.NewConstraint("^1")
	if err != nil {
		panic(err)
	}

	parser := evolviconf.NewParser[app.Configuration, *yaml.Decoder](
		evolviyaml.NewParser[app.Configuration, v1.YAMLConfiguration](
			constraint,
			v1.Changelog,
		),
	)

	parseAndPrint(
		parser,
		`
version: 1.0
host: localhost
port: 8080
authToken: "abc"`)

	// Output:
	// level=WARN msg="authToken is a field introduced in 1.1" line=5 column=1 field=authToken value=abc
	// {Host:localhost Port:8080}
}

func ExampleParseConfigWithDeprecatedFields() {
	constraint, err := semver.NewConstraint("^1")
	if err != nil {
		panic(err)
	}

	parser := evolviconf.NewParser[app.Configuration, *yaml.Decoder](
		evolviyaml.NewParser[app.Configuration, v1.YAMLConfiguration](
			constraint,
			v1.Changelog,
		),
	)

	parseAndPrint(
		parser,
		`
version: 1.2
host: localhost
port: 8080
authToken: "abc"`)

	// Output:
	// level=WARN msg="port is deprecated in 1.2, and will be removed in a future version" line=4 column=1 field=port value=8080
	// {Host:localhost Port:8080}
}

func parseAndPrint(parser *evolviconf.Parser[app.Configuration, *yaml.Decoder], yamlConf string) {
	reader := strings.NewReader(yamlConf)

	configs, warnings, err := parser.Parse(context.Background(), reader)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to parse YAML specification: %w", err))
	}

	if len(warnings) > 0 {
		warnings.Log(context.Background(), newLogger())
	}

	for _, config := range configs {
		fmt.Printf("%+v\n", config)
	}
}

func newLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(
		os.Stdout,
		&slog.HandlerOptions{ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			// Remove the time attribute, so the logs can be easily checked by the example test
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		}},
	))
}
