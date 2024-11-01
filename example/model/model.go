package model

type Configuration struct {
	Version   string
	Pipelines []Pipeline
}

type Pipeline struct {
	ID          string
	Status      string
	Name        string
	Description string
	Connectors  []Connector
	Processors  []Processor
	DLQ         DLQ
}

type Connector struct {
	ID         string
	Type       string
	Plugin     string
	Name       string
	Settings   map[string]string
	Processors []Processor
}

type Processor struct {
	ID        string
	Plugin    string
	Settings  map[string]string
	Workers   int
	Condition string
}

type DLQ struct {
	Plugin              string
	Settings            map[string]string
	WindowSize          *int
	WindowNackThreshold *int
}
