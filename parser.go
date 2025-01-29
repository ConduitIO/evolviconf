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

	"github.com/Masterminds/semver/v3"
)

type DecoderProvider[D any] interface {
	Decoder(io.Reader) D
}

type VersionParser[D any] interface {
	ParseVersion(ctx context.Context, decoder D) (*semver.Version, error)
}

type VersionedConfigParser[T, D any] interface {
	LatestKnownVersion() *semver.Version
	Constraint() *semver.Constraints
	ParseVersionedConfig(ctx context.Context, decoder D, version *semver.Version) (VersionedConfig[T], Warnings, error)
}

type AllInOneParser[T, D any] interface {
	DecoderProvider[D]
	VersionParser[D]
	VersionedConfigParser[T, D]
}

type VersionedConfig[T any] interface {
	ToConfig() (T, error)
}

type Parser[T, D any] struct {
	decoderProvider DecoderProvider[D]
	versionParser   VersionParser[D]
	configParsers   []VersionedConfigParser[T, D]
	latestVersion   *semver.Version
}

func NewParser[T, D any](
	allInOneParsers ...AllInOneParser[T, D],
) *Parser[T, D] {
	// Repack allInOneParsers into configParsers
	var firstParser AllInOneParser[T, D]
	configParsers := make([]VersionedConfigParser[T, D], len(allInOneParsers))
	for i, parser := range allInOneParsers {
		if firstParser == nil {
			firstParser = parser
		}
		configParsers[i] = parser
	}

	return NewParserExtended[T, D](
		firstParser,
		firstParser,
		configParsers...,
	)
}

func NewParserExtended[T, D any](
	decoderProvider DecoderProvider[D],
	versionParser VersionParser[D],
	configParsers ...VersionedConfigParser[T, D],
) *Parser[T, D] {
	latestVersion := semver.MustParse("0.0.0")
	for _, parser := range configParsers {
		if parser.LatestKnownVersion().GreaterThan(latestVersion) {
			latestVersion = parser.LatestKnownVersion()
		}
	}

	return &Parser[T, D]{
		decoderProvider: decoderProvider,
		versionParser:   versionParser,
		configParsers:   configParsers,
		latestVersion:   latestVersion,
	}
}

func (p *Parser[T, D]) Parse(ctx context.Context, reader io.Reader) ([]T, Warnings, error) {
	// we redirect everything read from reader to buffer with TeeReader, so that
	// we can first parse the version of the file and choose what type we
	// actually need to parse the configuration
	var buffer bytes.Buffer
	reader = io.TeeReader(reader, &buffer)

	versionDecoder := p.decoderProvider.Decoder(reader)
	configurationDecoder := p.decoderProvider.Decoder(&buffer)

	var configs []T
	var warnings Warnings

	for {
		version, w, err := p.parseVersion(ctx, versionDecoder)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, nil, err
		}
		warnings = append(warnings, w...)

		parser, perfectMatch := p.findVersionedConfigParser(version)
		if parser == nil {
			return nil, nil, fmt.Errorf("unsupported version %s", version)
		}

		if !perfectMatch {
			warnings = append(warnings, Warning{
				Message: fmt.Sprintf("no parser found for version %s, using parser for version %s with costraints %s", version, parser.LatestKnownVersion(), parser.Constraint()),
			})
		}

		config, w, err := parser.ParseVersionedConfig(ctx, configurationDecoder, version)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to parse versioned config: %w", err)
		}
		warnings = append(warnings, w.Sort()...)

		out, err := config.ToConfig()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to convert versioned config to actual config: %w", err)
		}

		configs = append(configs, out)
	}

	return configs, warnings, nil
}

func (p *Parser[T, D]) parseVersion(ctx context.Context, decoder D) (*semver.Version, Warnings, error) {
	version, err := p.versionParser.ParseVersion(ctx, decoder)
	if err != nil {
		if errors.Is(err, ErrVersionNotSpecified) {
			// No version specified, fall back to the latest known version.
			return p.latestVersion, Warnings{{
				Message: "no version defined, falling back to parser version " + p.latestVersion.String(),
			}}, nil
		}
		return nil, nil, fmt.Errorf("failed to parse version: %w", err)
	}

	return version, nil, nil
}

// findVersionedConfigParser returns the versioned config parser for the version
// and a boolean denoting if it's a perfect match. If it's not a perfect match,
// the best possible match is returned.
func (p *Parser[T, D]) findVersionedConfigParser(version *semver.Version) (VersionedConfigParser[T, D], bool) {
	var bestMatch VersionedConfigParser[T, D]
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
