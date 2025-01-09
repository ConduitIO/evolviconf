# EvolviConf

[![License](https://img.shields.io/badge/license-Apache%202-blue)](/LICENSE.md)

EvolviConf is a minimalistic Go library that handles versioned (evolving)
configuration files.

A single `evolviconf.Parser` can read different versions of a configuration
object(s) found in a file(s), print information about changes (field
deprecated/introduced), warn about unknown fields, fall back to a version, etc.

EvolviConf itself can handle any file type as long as there's a parser that
implements
the [evolviconf.AllInOneParser](https://github.com/ConduitIO/evolviconf/blob/83c36707434f4f3121d83f282acaf402ec617b11/parser.go#L41)
interface. Currently, we have
a [YAML parser](https://github.com/ConduitIO/evolviconf/tree/main/evolviyaml).

Examples of using EvolviConf can be found in the [examples](/examples)
directory.

EvolviConf was created and open-sourced by [Meroxa](https://meroxa.io).