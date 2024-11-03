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
	"github.com/conduitio/evolviconf/evolviyaml/example/yaml/model"
)

// Changelog should be adjusted every time we change the pipeline config and add
// a new config version. Based on the changelog the parser will output warnings.
var Changelog = evolviconf.Changelog{
	semver.MustParse("1.0"): {{ // deprecate fields in version 1.0 so a warning is logged for all v1 pipeline configs
		Field:      "pipelines.*.processors",
		ChangeType: evolviconf.FieldDeprecated,
		Message:    "the order of processors is non-deterministic in configuration files with version 1.x, please upgrade to version 2.x",
	}, {
		Field:      "pipelines.*.connectors.*.processors",
		ChangeType: evolviconf.FieldDeprecated,
		Message:    "the order of processors is non-deterministic in configuration files with version 1.x, please upgrade to version 2.x",
	}},
	semver.MustParse("1.1"): {{
		Field:      "pipelines.*.dead-letter-queue",
		ChangeType: evolviconf.FieldIntroduced,
		Message:    "field dead-letter-queue was introduced in version 1.1, please update the pipeline config version",
	}},
}

type Configuration struct {
	Version   string              `yaml:"version" json:"version"`
	Pipelines map[string]Pipeline `yaml:"pipelines" json:"pipelines"`
}

type Pipeline struct {
	Status      string               `yaml:"status" json:"status"`
	Name        string               `yaml:"name" json:"name"`
	Description string               `yaml:"description" json:"description"`
	Connectors  map[string]Connector `yaml:"connectors,omitempty" json:"connectors,omitempty"`
	Processors  map[string]Processor `yaml:"processors,omitempty" json:"processors,omitempty"`
	DLQ         DLQ                  `yaml:"dead-letter-queue" json:"dead-letter-queue"`
}

type Connector struct {
	Type       string               `yaml:"type" json:"type"`
	Plugin     string               `yaml:"plugin" json:"plugin"`
	Name       string               `yaml:"name" json:"name"`
	Settings   map[string]string    `yaml:"settings" json:"settings"`
	Processors map[string]Processor `yaml:"processors,omitempty" json:"processors,omitempty"`
}

type Processor struct {
	Type     string            `yaml:"type" json:"type"`
	Settings map[string]string `yaml:"settings" json:"settings"`
	Workers  int               `yaml:"workers" json:"workers"`
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
		i := 0
		for id, pipeline := range c.Pipelines {
			cfg.Pipelines[i] = pipeline.ToConfig()
			cfg.Pipelines[i].ID = id
			i++
		}
	}
	return cfg
}

func (p Pipeline) ToConfig() model.Pipeline {
	return model.Pipeline{
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
	connectors := make([]model.Connector, 0, len(p.Connectors))
	for id, connector := range p.Connectors {
		c := connector.ToConfig()
		c.ID = id
		connectors = append(connectors, c)
	}
	return connectors
}

func (p Pipeline) processorsToConfig() []model.Processor {
	if len(p.Processors) == 0 {
		return nil
	}
	processors := make([]model.Processor, 0, len(p.Processors))

	// Warning: this ordering is not deterministic, v2 of the pipeline config
	// fixes this.
	for id, processor := range p.Processors {
		proc := processor.ToConfig()
		proc.ID = id
		processors = append(processors, proc)
	}
	return processors
}

func (c Connector) ToConfig() model.Connector {
	return model.Connector{
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
	processors := make([]model.Processor, 0, len(c.Processors))

	// Warning: this ordering is not deterministic, v2 of the pipeline config
	// fixes this.
	for id, processor := range c.Processors {
		proc := processor.ToConfig()
		proc.ID = id
		processors = append(processors, proc)
	}
	return processors
}

func (p Processor) ToConfig() model.Processor {
	return model.Processor{
		// Type was removed in favor of Plugin
		Plugin:   p.Type,
		Settings: p.Settings,
		Workers:  p.Workers,
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
