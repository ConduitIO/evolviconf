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
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/yaml/v3"
)

type Parser[T any, C evolviconf.VersionedConfig[T]] struct {
	constraint         *semver.Constraints
	latestKnownVersion *semver.Version
	linter             *configLinter
}

func NewParser[T any, C evolviconf.VersionedConfig[T]](
	constraint *semver.Constraints,
	latestKnownVersion *semver.Version,
	changelog evolviconf.Changelog,
) *Parser[T, C] {
	return &Parser[T, C]{
		constraint:         constraint,
		latestKnownVersion: latestKnownVersion,
		linter:             newConfigLinter(changelog),
	}
}

func (p *Parser[T, C]) LatestKnownVersion() *semver.Version {
	return p.latestKnownVersion
}

func (p *Parser[T, C]) Constraint() *semver.Constraints {
	return p.constraint
}

func (p *Parser[T, C]) ParseVersion(_ context.Context, reader io.Reader) (*semver.Version, evolviconf.Position, error) {
	dec := yaml.NewDecoder(reader)

	var out struct {
		Version string `yaml:"version"`
	}

	// versionNode will store the node that contains the version field (for warning)
	var versionNode yaml.Node
	dec.WithHook(func(path []string, node *yaml.Node) {
		if len(path) == 1 && path[0] == "version" {
			versionNode = *node
		}
	})

	err := dec.Decode(&out)
	if err != nil {
		return nil, evolviconf.Position{}, err
	}

	pos := evolviconf.Position{
		Field:  "version",
		Line:   versionNode.Line,
		Column: versionNode.Column,
		Value:  versionNode.Value,
	}

	if out.Version == "" {
		return nil, pos, evolviconf.ErrVersionNotSpecified
	}

	version, err := semver.NewVersion(out.Version)
	return version, pos, err
}

func (p *Parser[T, C]) ParseVersionedConfig(_ context.Context, reader io.Reader, version *semver.Version) (evolviconf.VersionedConfig[T], evolviconf.Warnings, error) {
	dec := yaml.NewDecoder(reader)

	// set up decoder hooks
	var warn evolviconf.Warnings
	dec.KnownFields(true)
	dec.WithHook(multiDecoderHook(
		envDecoderHook, // replace environment variables with their values
		p.linter.DecoderHook(version.String(), &warn), // lint config as it's parsed
	))

	cfg := zero[C]()
	err := dec.Decode(&cfg)

	if err != nil {
		// check if it's a type error (document was partially decoded)
		var typeErr *yaml.TypeError
		if errors.As(err, &typeErr) {
			var w evolviconf.Warnings
			w, err = p.yamlTypeErrorToWarnings(typeErr)
			warn = append(warn, w...)
		}
		// check if we recovered from the error
		if err != nil {
			return zero[C](), nil, fmt.Errorf("decoding error: %w", err)
		}
	}

	return cfg, warn, nil
}

// yamlTypeErrorToWarnings converts yaml.TypeError to warnings if it only
// contains recoverable errors. If it contains at least one actual error it
// returns nil and the error itself.
func (p *Parser[T, C]) yamlTypeErrorToWarnings(typeErr *yaml.TypeError) (evolviconf.Warnings, error) {
	warn := make(evolviconf.Warnings, len(typeErr.Errors))
	for i, uerr := range typeErr.Errors {
		switch uerr := uerr.(type) {
		case *yaml.UnknownFieldError:
			warn[i] = evolviconf.Warning{
				Position: evolviconf.Position{
					Field:  uerr.Field(),
					Line:   uerr.Line(),
					Column: uerr.Column(),
					Value:  "", // no value in UnknownFieldError
				},
				Message: uerr.Error(),
			}
		default:
			// we don't tolerate any other errors
			return nil, typeErr
		}
	}
	return warn, nil
}

func zero[T any]() T {
	var t T
	return t
}
