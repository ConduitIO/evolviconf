# EvolviConf

[![License](https://img.shields.io/badge/license-Apache%202-blue)](/LICENSE.md)

EvolviConf is a Go library that handles versioned (evolving) configuration
files.

A single `evolviconf.Parser` can handle multiple versions of a configuration
file, log warnings about unknown fields, fall back to a version, etc.

EvolviConf supports 