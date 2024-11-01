module github.com/conduitio/evolviconf/example/yaml

go 1.23.0

require (
	github.com/Masterminds/semver/v3 v3.3.0
	github.com/conduitio/evolviconf v0.0.0
	github.com/conduitio/evolviconf/evolviyaml v0.0.0
	github.com/goccy/go-json v0.10.3
	github.com/google/go-cmp v0.6.0
	github.com/matryer/is v1.4.1
)

require github.com/conduitio/yaml/v3 v3.3.0 // indirect

replace github.com/conduitio/evolviconf => ../../

replace github.com/conduitio/evolviconf/evolviyaml => ../../evolviyaml
