> ## Config
>
> This is a new library aiming to simplify runtime configuration for Go
> applications. Expect more updates shortly.
>


[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/activatedio/config/ci.yaml?branch=main&style=flat-square)](https://github.com/activatedio/config/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/activatedio/config?style=flat-square)](https://goreportcard.com/report/github.com/activatedio/config)
![Go Version](https://img.shields.io/github/go-mod/go-version/activatedio/config?style=flat-square)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/activatedio/config)](https://pkg.go.dev/mod/github.com/activatedio/config)

# Config

Runtime library for flexible configuration.

## Install

``` sh
go get -u github.com/activatedio/config

```

## Usage

The following example shows how to create a new config, add sources, and retrieve values

``` go

cfg := config.New()

cfg.AddSource(config.FromYAMLFile("config.yaml"))
// Can be strings, maps or structs
cfg.AddSource(config.FromValue("prefix.key", "value"))
cfg.AddLateBindingSource(config.FromEnvironmentVars())

// Read reads value
var val *string
err := config.Read("prefix.key", val)

// MustRead does the same but panics on error
//var val *string
//config.MustRead("prefix.key", val)


fmt.Println(val)

```
