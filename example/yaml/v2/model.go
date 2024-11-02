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

package v2

import (
	"github.com/Masterminds/semver/v3"
	"github.com/conduitio/evolviconf"
	"github.com/conduitio/evolviconf/example/yaml/model"
)

// Changelog should be adjusted every time we change the pipeline config and add
// a new config version. Based on the changelog the parser will output warnings.
var Changelog = evolviconf.Changelog{
	semver.MustParse("2.0"): {}, // initial version
	semver.MustParse("2.1"): {
		{
			Field:      "pipelines.*.processors.*.condition",
			ChangeType: evolviconf.FieldIntroduced,
			Message:    "field condition was introduced in version 2.1, please update the pipeline config version",
		},
		{
			Field:      "pipelines.*.connectors.*.processors.*.condition",
			ChangeType: evolviconf.FieldIntroduced,
			Message:    "field condition was introduced in version 2.1, please update the pipeline config version",
		},
	},
	semver.MustParse("2.2"): {
		{
			Field:      "pipelines.*.processors.*.plugin",
			ChangeType: evolviconf.FieldIntroduced,
			Message:    "field plugin was introduced in version 2.2, please update the pipeline config version",
		},
		{
			Field:      "pipelines.*.connectors.*.processors.*.plugin",
			ChangeType: evolviconf.FieldIntroduced,
			Message:    "field plugin was introduced in version 2.2, please update the pipeline config version",
		},
		{
			Field:      "pipelines.*.processors.*.type",
			ChangeType: evolviconf.FieldDeprecated,
			Message:    "please use field 'plugin' (introduced in version 2.2)",
		},
		{
			Field:      "pipelines.*.connectors.*.processors.*.type",
			ChangeType: evolviconf.FieldDeprecated,
			Message:    "please use field 'plugin' (introduced in version 2.2)",
		},
	},
}

type Configuration struct {
	Version   string     `yaml:"version" json:"version"`
	Pipelines []Pipeline `yaml:"pipelines" json:"pipelines"`
}

type Pipeline struct {
	ID          string      `yaml:"id" json:"id"`
	Status      string      `yaml:"status" json:"status"`
	Name        string      `yaml:"name" json:"name"`
	Description string      `yaml:"description" json:"description"`
	Connectors  []Connector `yaml:"connectors" json:"connectors"`
	Processors  []Processor `yaml:"processors" json:"processors"`
	DLQ         DLQ         `yaml:"dead-letter-queue" json:"dead-letter-queue"`
}

type Connector struct {
	ID         string            `yaml:"id" json:"id"`
	Type       string            `yaml:"type" json:"type"`
	Plugin     string            `yaml:"plugin" json:"plugin"`
	Name       string            `yaml:"name" json:"name"`
	Settings   map[string]string `yaml:"settings" json:"settings"`
	Processors []Processor       `yaml:"processors" json:"processors"`
}

type Processor struct {
	ID        string            `yaml:"id" json:"id"`
	Type      string            `yaml:"type" json:"type"`
	Plugin    string            `yaml:"plugin" json:"plugin"`
	Condition string            `yaml:"condition" json:"condition"`
	Settings  map[string]string `yaml:"settings" json:"settings"`
	Workers   int               `yaml:"workers" json:"workers"`
}

type DLQ struct {
	Plugin              string            `yaml:"plugin" json:"plugin"`
	Settings            map[string]string `yaml:"settings" json:"settings"`
	WindowSize          *int              `yaml:"window-size" json:"window-size"`
	WindowNackThreshold *int              `yaml:"window-nack-threshold" json:"window-nack-threshold"`
}

func (c Configuration) ToConfig() model.Configuration {
	cfg := model.Configuration{Version: c.Version}
	if len(c.Pipelines) > 0 {
		cfg.Pipelines = make([]model.Pipeline, len(c.Pipelines))
		for i, pipeline := range c.Pipelines {
			cfg.Pipelines[i] = pipeline.ToConfig()
		}
	}
	return cfg
}

func (p Pipeline) ToConfig() model.Pipeline {
	return model.Pipeline{
		ID:          p.ID,
		Status:      p.Status,
		Name:        p.Name,
		Description: p.Description,
		Connectors:  p.connectorsToConfig(),
		Processors:  p.processorsToConfig(),
		DLQ:         p.DLQ.ToConfig(),
	}
}

func (p Pipeline) connectorsToConfig() []model.Connector {
	if len(p.Connectors) == 0 {
		return nil
	}
	connectors := make([]model.Connector, len(p.Connectors))
	for i, connector := range p.Connectors {
		connectors[i] = connector.ToConfig()
	}
	return connectors
}

func (p Pipeline) processorsToConfig() []model.Processor {
	if len(p.Processors) == 0 {
		return nil
	}
	processors := make([]model.Processor, len(p.Processors))

	for i, processor := range p.Processors {
		processors[i] = processor.ToConfig()
	}
	return processors
}

func (c Connector) ToConfig() model.Connector {
	return model.Connector{
		ID:         c.ID,
		Type:       c.Type,
		Plugin:     c.Plugin,
		Name:       c.Name,
		Settings:   c.Settings,
		Processors: c.processorsToConfig(),
	}
}

func (c Connector) processorsToConfig() []model.Processor {
	if len(c.Processors) == 0 {
		return nil
	}
	processors := make([]model.Processor, len(c.Processors))

	for i, processor := range c.Processors {
		processors[i] = processor.ToConfig()
	}
	return processors
}

func (p Processor) ToConfig() model.Processor {
	plugin := p.Plugin
	if plugin == "" {
		plugin = p.Type
	}

	return model.Processor{
		ID:        p.ID,
		Plugin:    plugin,
		Settings:  p.Settings,
		Workers:   p.Workers,
		Condition: p.Condition,
	}
}

func (p DLQ) ToConfig() model.DLQ {
	return model.DLQ{
		Plugin:              p.Plugin,
		Settings:            p.Settings,
		WindowSize:          p.WindowSize,
		WindowNackThreshold: p.WindowNackThreshold,
	}
}
