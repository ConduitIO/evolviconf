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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/Masterminds/semver/v3"
)

type VersionParser interface {
	ParseVersion(ctx context.Context, reader io.Reader) (*semver.Version, Position, error)
}

type VersionParserFunc func(ctx context.Context, reader io.Reader) (*semver.Version, Position, error)

func (f VersionParserFunc) ParseVersion(ctx context.Context, reader io.Reader) (*semver.Version, Position, error) {
	return f(ctx, reader)
}

type VersionedConfigParser[T any] interface {
	LatestKnownVersion() *semver.Version
	Constraint() *semver.Constraints
	ParseVersionedConfig(ctx context.Context, reader io.Reader, version *semver.Version) (VersionedConfig[T], Warnings, error)
}

type VersionedConfig[T any] interface {
	ToConfig() T
}

type Parser[T any] struct {
	logger *slog.Logger

	versionParser VersionParser
	configParsers []VersionedConfigParser[T]
	latestVersion *semver.Version
}

func NewParser[T any](
	logger *slog.Logger,
	versionParser VersionParser,
	configParsers []VersionedConfigParser[T],
) (*Parser[T], error) {
	latestVersion := semver.MustParse("0.0.0")
	for _, parser := range configParsers {
		if parser.LatestKnownVersion().GreaterThan(latestVersion) {
			latestVersion = parser.LatestKnownVersion()
		}
	}

	return &Parser[T]{
		logger: logger,

		versionParser: versionParser,
		configParsers: configParsers,
		latestVersion: latestVersion,
	}, nil
}

func (p *Parser[T]) Parse(ctx context.Context, reader io.Reader) (T, Warnings, error) {
	// we redirect everything read from reader to buffer with TeeReader, so that
	// we can first parse the version of the file and choose what type we
	// actually need to parse the configuration
	var buffer bytes.Buffer
	reader = io.TeeReader(reader, &buffer)

	version, warnings, err := p.parseVersion(ctx, reader)
	if err != nil {
		return zero[T](), nil, err
	}

	parser, perfectMatch := p.findVersionedConfigParser(version)
	if parser == nil {
		return zero[T](), nil, fmt.Errorf("unsupported version %s", version)
	}

	if !perfectMatch {
		warnings = append(warnings, Warning{
			Position: Position{},
			Message:  fmt.Sprintf("no parser found for version %s, using parser for version %s with costraints %s", version, parser.LatestKnownVersion(), parser.Constraint()),
		})
	}

	config, w, err := parser.ParseVersionedConfig(ctx, reader, version)
	if err != nil {
		return zero[T](), nil, err
	}
	warnings = append(warnings, w...)

	return config.ToConfig(), warnings, nil
}

func (p *Parser[T]) parseVersion(ctx context.Context, reader io.Reader) (*semver.Version, Warnings, error) {
	version, pos, err := p.versionParser.ParseVersion(ctx, reader)
	if err != nil {
		if errors.Is(err, ErrVersionNotSpecified) {
			// No version specified, fall back to the latest known version.
			return p.latestVersion, Warnings{{
				Position: pos,
				Message:  "no version defined, falling back to parser version " + p.latestVersion.String(),
			}}, nil
		}
		return nil, nil, err
	}

	return version, nil, nil
}

// findVersionedConfigParser returns the versioned config parser for the version
// and a boolean denoting if it's a perfect match. If it's not a perfect match,
// the best possible match is returned.
func (p *Parser[T]) findVersionedConfigParser(version *semver.Version) (VersionedConfigParser[T], bool) {
	var bestMatch VersionedConfigParser[T]
	for _, parser := range p.configParsers {
		if parser.Constraint().Check(version) {
			if parser.LatestKnownVersion().GreaterThanEqual(version) {
				// This is a perfect match.
				return parser, true
			}

			// The constraint is satisfied, but the latest known version is smaller,
			// store this as the best match and continue searching if there is a
			// better parser in the list.
			if bestMatch == nil || bestMatch.LatestKnownVersion().LessThan(parser.LatestKnownVersion()) {
				bestMatch = parser
			}
		}
	}
	return bestMatch, false
}

func zero[T any]() T {
	var t T
	return t
}
