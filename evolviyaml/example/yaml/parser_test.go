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

package yaml

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/evolviconf/evolviyaml"
	"github.com/conduitio/evolviconf/evolviyaml/example/yaml/model"
	v1 "github.com/conduitio/evolviconf/evolviyaml/example/yaml/v1"
	v2 "github.com/conduitio/evolviconf/evolviyaml/example/yaml/v2"
	"github.com/conduitio/yaml/v3"
	"github.com/google/go-cmp/cmp"
	"github.com/matryer/is"
)

func must[T any](out T, err error) T {
	if err != nil {
		panic(err)
	}
	return out
}

func TestParser_V1_Success(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()

	filepath := "./v1/testdata/pipelines1-success.yml"
	intPtr := func(i int) *int { return &i }
	want := []model.Configuration{
		{
			Version: "1.0",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline1",
					Status:      "running",
					Name:        "pipeline1",
					Description: "desc1",
					Processors: []model.Processor{
						{
							ID:     "pipeline1proc1",
							Plugin: "js",
							Settings: map[string]string{
								"additionalProp1": "string",
								"additionalProp2": "string",
							},
						},
					},
					Connectors: []model.Connector{
						{
							ID:     "con1",
							Type:   "source",
							Plugin: "builtin:s3",
							Name:   "s3-source",
							Settings: map[string]string{
								"aws.region": "us-east-1",
								"aws.bucket": "my-bucket",
							},
							Processors: []model.Processor{
								{
									ID:     "proc1",
									Plugin: "js",
									Settings: map[string]string{
										"additionalProp1": "string",
										"additionalProp2": "string",
									},
								},
							},
						},
					},
					DLQ: model.DLQ{
						Plugin: "my-plugin",
						Settings: map[string]string{
							"foo": "bar",
						},
						WindowSize:          intPtr(4),
						WindowNackThreshold: intPtr(2),
					},
				},
			},
		},
		{
			Version: "1.12",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline2",
					Status:      "stopped",
					Name:        "pipeline2",
					Description: "desc2",
					Connectors: []model.Connector{
						{
							ID:     "con2",
							Type:   "destination",
							Plugin: "builtin:file",
							Name:   "file-dest",
							Settings: map[string]string{
								"path": "my/path",
							},
							Processors: []model.Processor{
								{
									ID:     "con2proc1",
									Plugin: "hoistfield",
									Settings: map[string]string{
										"additionalProp1": "string",
										"additionalProp2": "string",
									},
								},
							},
						},
					},
				},
				{
					ID:          "pipeline3",
					Status:      "stopped",
					Name:        "pipeline3",
					Description: "empty",
				},
			},
		},
	}

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)

	is.NoErr(err)
	is.Equal("", cmp.Diff(got, want))
}

func TestParser_V1_Warnings(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()

	filepath := "./v1/testdata/pipelines1-success.yml"
	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	_, warnings, err := parser.Parse(context.Background(), file)
	is.NoErr(err)

	// check warnings
	want := `{"level":"WARN","msg":"field unknownField not found in type v1.Pipeline","line":5,"column":5,"field":"unknownField"}
{"level":"WARN","msg":"the order of processors is non-deterministic in configuration files with version 1.x, please upgrade to version 2.x","line":17,"column":9,"field":"processors"}
{"level":"WARN","msg":"the order of processors is non-deterministic in configuration files with version 1.x, please upgrade to version 2.x","line":23,"column":5,"field":"processors"}
{"level":"WARN","msg":"field dead-letter-queue was introduced in version 1.1, please update the pipeline config version","line":30,"column":5,"field":"dead-letter-queue"}
{"level":"WARN","msg":"no parser found for version 1.12.0, using parser for version 1.1.0 with costraints ^1"}
{"level":"WARN","msg":"the order of processors is non-deterministic in configuration files with version 1.x, please upgrade to version 2.x","line":51,"column":9,"field":"processors"}
`

	var out bytes.Buffer
	logger := bufferLogger(&out)
	warnings.Log(context.Background(), logger)

	is.Equal(out.String(), want)
}

func TestParser_V1_EmptyFile(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v1/testdata/pipelines3-empty.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal(got, nil)
}

func TestParser_V1_InvalidYaml(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v1/testdata/pipelines4-invalid-yaml.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	_, _, err = parser.Parse(context.Background(), file)
	is.True(err != nil)
}

func TestParser_V1_EnvVars(t *testing.T) {
	is := is.New(t)
	parser := newTestParser(evolviyaml.EnvDecoderHook)
	filepath := "./v1/testdata/pipelines5-env-vars.yml"

	// set env variables
	t.Setenv("TEST_PARSER_AWS_SECRET", "my-aws-secret")
	t.Setenv("TEST_PARSER_AWS_KEY", "my-aws-key")
	t.Setenv("TEST_PARSER_AWS_URL", "aws-url")

	want := []model.Configuration{
		{
			Version: "1.0",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline1",
					Status:      "running",
					Name:        "pipeline1",
					Description: "desc1",
					Connectors: []model.Connector{
						{
							ID:     "con1",
							Type:   "source",
							Plugin: "builtin:s3",
							Name:   "s3-source",
							Settings: map[string]string{
								// env variables should be replaced with their values
								"aws.secret": "my-aws-secret",
								"aws.key":    "my-aws-key",
								"aws.url":    "my/aws-url/url",
							},
						},
					},
				},
			},
		},
	}

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal(got, want)
}

func TestParser_V1_ParseV2Config(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v2/testdata/pipelines1-success.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	// replace major version so that the v1 parser is chosen for a v2 config
	r := replacingReader{
		Reader: file,
		Old:    []byte("version: 2"),
		New:    []byte("version: 1"),
	}

	_, _, err = parser.Parse(context.Background(), r)
	is.True(err != nil)
	// make sure it's an invalid type error
	var iterr *yaml.InvalidTypeError
	is.True(errors.As(err, &iterr))
}

func TestParser_V2_Success(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v2/testdata/pipelines1-success.yml"
	intPtr := func(i int) *int { return &i }
	want := []model.Configuration{
		{
			Version: "2.2",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline1",
					Status:      "running",
					Name:        "pipeline1",
					Description: "desc1",
					Processors: []model.Processor{
						{
							ID:     "pipeline1proc1",
							Plugin: "js",
							Settings: map[string]string{
								"additionalProp1": "string",
								"additionalProp2": "string",
							},
						},
					},
					Connectors: []model.Connector{
						{
							ID:     "con1",
							Type:   "source",
							Plugin: "builtin:s3",
							Name:   "s3-source",
							Settings: map[string]string{
								"aws.region": "us-east-1",
								"aws.bucket": "my-bucket",
							},
							Processors: []model.Processor{
								{
									ID:     "proc1",
									Plugin: "js",
									Settings: map[string]string{
										"additionalProp1": "string",
										"additionalProp2": "string",
									},
								},
							},
						},
					},
					DLQ: model.DLQ{
						Plugin: "my-plugin",
						Settings: map[string]string{
							"foo": "bar",
						},
						WindowSize:          intPtr(4),
						WindowNackThreshold: intPtr(2),
					},
				},
			},
		},
		model.Configuration{
			Version: "2.12",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline2",
					Status:      "stopped",
					Name:        "pipeline2",
					Description: "desc2",
					Connectors: []model.Connector{
						{
							ID:     "con2",
							Type:   "destination",
							Plugin: "builtin:file",
							Name:   "file-dest",
							Settings: map[string]string{
								"path": "my/path",
							},
							Processors: []model.Processor{
								{
									ID:     "con2proc1",
									Plugin: "hoistfield",
									Settings: map[string]string{
										"additionalProp1": "string",
										"additionalProp2": "string",
									},
								},
							},
						},
					},
				},
				{
					ID:          "pipeline3",
					Status:      "stopped",
					Name:        "pipeline3",
					Description: "empty",
				},
			},
		},
	}

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal("", cmp.Diff(got, want))
}

func TestParser_V2_BackwardsCompatibility(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v2/testdata/pipelines6-bwc.yml"
	want := []model.Configuration{
		{
			Version: "2.2",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline6",
					Status:      "running",
					Name:        "pipeline6",
					Description: "desc1",
					Processors: []model.Processor{
						{
							ID:     "pipeline1proc1",
							Plugin: "js",
							Settings: map[string]string{
								"additionalProp1": "string",
								"additionalProp2": "string",
							},
						},
					},
					Connectors: []model.Connector{
						{
							ID:     "con1",
							Type:   "source",
							Plugin: "builtin:s3",
							Name:   "s3-source",
							Settings: map[string]string{
								"aws.region": "us-east-1",
								"aws.bucket": "my-bucket",
							},
							Processors: []model.Processor{
								{
									ID:     "proc1",
									Plugin: "js",
									Settings: map[string]string{
										"additionalProp1": "string",
										"additionalProp2": "string",
									},
								},
							},
						},
					},
				},
			},
		},
	}

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal("", cmp.Diff(want, got))
}

func TestParser_V2_Warnings(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()

	filepath := "./v2/testdata/pipelines1-success.yml"
	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	_, warnings, err := parser.Parse(context.Background(), file)
	is.NoErr(err)

	// check warnings
	want := `{"level":"WARN","msg":"field unknownField not found in type v2.Pipeline","line":6,"column":5,"field":"unknownField"}
{"level":"WARN","msg":"no parser found for version 2.12.0, using parser for version 2.2.0 with costraints ^2"}
`

	var out bytes.Buffer
	logger := bufferLogger(&out)
	warnings.Log(context.Background(), logger)

	is.Equal(out.String(), want)
}

func TestParser_V2_EmptyFile(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v2/testdata/pipelines3-empty.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal(got, nil)
}

func TestParser_V2_InvalidYaml(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v2/testdata/pipelines4-invalid-yaml.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	_, _, err = parser.Parse(context.Background(), file)
	is.True(err != nil)
}

func TestParser_V2_EnvVars(t *testing.T) {
	is := is.New(t)
	parser := newTestParser(evolviyaml.EnvDecoderHook)
	filepath := "./v2/testdata/pipelines5-env-vars.yml"

	// set env variables
	t.Setenv("TEST_PARSER_AWS_SECRET", "my-aws-secret")
	t.Setenv("TEST_PARSER_AWS_KEY", "my-aws-key")
	t.Setenv("TEST_PARSER_AWS_URL", "aws-url")

	want := []model.Configuration{
		{
			Version: "2.0",
			Pipelines: []model.Pipeline{
				{
					ID:          "pipeline1",
					Status:      "running",
					Name:        "pipeline1",
					Description: "desc1",
					Connectors: []model.Connector{
						{
							ID:     "con1",
							Type:   "source",
							Plugin: "builtin:s3",
							Name:   "s3-source",
							Settings: map[string]string{
								// env variables should be replaced with their values
								"aws.secret": "my-aws-secret",
								"aws.key":    "my-aws-key",
								"aws.url":    "my/aws-url/url",
							},
						},
					},
				},
			},
		},
	}

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	got, _, err := parser.Parse(context.Background(), file)
	is.NoErr(err)
	is.Equal(got, want)
}

func TestParser_V2_ParseV1Config(t *testing.T) {
	is := is.New(t)
	parser := newTestParser()
	filepath := "./v1/testdata/pipelines1-success.yml"

	file, err := os.Open(filepath)
	is.NoErr(err)
	defer file.Close()

	// replace major version so that the v2 parser is chosen for a v1 config
	r := replacingReader{
		Reader: file,
		Old:    []byte("version: 1"),
		New:    []byte("version: 2"),
	}

	_, _, err = parser.Parse(context.Background(), r)
	is.True(err != nil)
	// make sure it's an invalid type error
	var iterr *yaml.InvalidTypeError
	is.True(errors.As(err, &iterr))
}

// replacingReader wraps a reader and replaces Old with New while reading.
type replacingReader struct {
	io.Reader
	Old []byte
	New []byte
}

func (rr replacingReader) Read(p []byte) (int, error) {
	i, err := rr.Reader.Read(p)
	if err != nil {
		return i, err
	}
	// that's very naive, Read reads up to len(p) bytes, so it could happen that
	// the sequence we are looking for is split in two
	// we don't care, it's good enough for our tests
	tmp := bytes.ReplaceAll(p, rr.Old, rr.New)
	copy(p, tmp)
	return i, nil
}

func newTestParser(hooks ...yaml.DecoderHook) *evolviconf.Parser[model.Configuration, *yaml.Decoder] {
	v1Parser := evolviyaml.NewParser[model.Configuration, v1.Configuration](
		must[*semver.Constraints](semver.NewConstraint("^1")),
		v1.Changelog,
	)
	v2Parser := evolviyaml.NewParser[model.Configuration, v2.Configuration](
		must[*semver.Constraints](semver.NewConstraint("^2")),
		v2.Changelog,
	)
	v1Parser.WithHook(evolviyaml.MultiDecoderHook(hooks...))
	v2Parser.WithHook(evolviyaml.MultiDecoderHook(hooks...))
	return evolviconf.NewParser(v1Parser, v2Parser)
}

func bufferLogger(out io.Writer) *slog.Logger {
	return slog.New(slog.NewJSONHandler(out, &slog.HandlerOptions{
		ReplaceAttr: func(_ []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				return slog.Attr{}
			}
			return a
		},
	}))
}
