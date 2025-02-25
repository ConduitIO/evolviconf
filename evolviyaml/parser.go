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

package evolviyaml

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"sort"

	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/yaml/v3"
)

type Parser[T any, C evolviconf.VersionedConfig[T]] struct {
	constraint         *semver.Constraints
	latestKnownVersion *semver.Version
	linter             *configLinter
	hook               yaml.DecoderHook
}

func NewParser[T any, C evolviconf.VersionedConfig[T]](
	constraint *semver.Constraints,
	changelog evolviconf.Changelog,
) *Parser[T, C] {
	var versions semver.Collection
	for k := range maps.Keys(changelog) {
		versions = append(versions, k)
	}
	sort.Sort(versions)

	return &Parser[T, C]{
		constraint:         constraint,
		latestKnownVersion: versions[len(versions)-1],
		linter:             newConfigLinter(changelog),
	}
}

func (p *Parser[T, C]) WithHook(hook yaml.DecoderHook) *Parser[T, C] {
	p.hook = hook
	return p
}

func (p *Parser[T, C]) Decoder(reader io.Reader) *yaml.Decoder {
	return yaml.NewDecoder(reader)
}

func (p *Parser[T, C]) LatestKnownVersion() *semver.Version {
	return p.latestKnownVersion
}

func (p *Parser[T, C]) Constraint() *semver.Constraints {
	return p.constraint
}

func (p *Parser[T, C]) ParseVersion(_ context.Context, dec *yaml.Decoder) (*semver.Version, error) {
	var out struct {
		Version string `yaml:"version"`
	}

	err := dec.Decode(&out)
	if err != nil {
		return nil, err
	}

	if out.Version == "" {
		return nil, evolviconf.ErrVersionNotSpecified
	}

	version, err := semver.NewVersion(out.Version)
	return version, err
}

func (p *Parser[T, C]) ParseVersionedConfig(_ context.Context, dec *yaml.Decoder, version *semver.Version) (evolviconf.VersionedConfig[T], evolviconf.Warnings, error) {
	// set up decoder hooks
	var warn evolviconf.Warnings
	dec.KnownFields(true)
	dec.WithHook(MultiDecoderHook(
		p.hook,
		p.linter.DecoderHook(version, &warn), // lint config as it's parsed
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
