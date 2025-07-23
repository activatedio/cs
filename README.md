> ## Config
>
> This is a new library aiming to simplify runtime csuration for Go
> applications. Expect more updates shortly.
>


[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/activatedio/cs/ci.yaml?branch=main&style=flat-square)](https://github.com/activatedio/cs/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/activatedio/cs?style=flat-square)](https://goreportcard.com/report/github.com/activatedio/cs)
![Go Version](https://img.shields.io/github/go-mod/go-version/activatedio/cs?style=flat-square)
[![PkgGoDev](https://pkg.go.dev/badge/mod/github.com/activatedio/cs)](https://pkg.go.dev/mod/github.com/activatedio/cs)

# Config

Runtime library for flexible csuration.

## Install

``` sh
go get -u github.com/activatedio/cs

```

## Usage

The following example shows how to create a new cs, add sources, and retrieve values

``` go

cfg := cs.New()

cfg.AddSource(cs.FromYAMLFile("cs.yaml"))
// Can be strings, maps or structs
cfg.AddSource(cs.FromValue("prefix.key", "value"))
cfg.AddLateBindingSource(cs.FromEnvironmentVars())

// Read reads value
var val *string
err := cs.Read("prefix.key", val)

// MustRead does the same but panics on error
//var val *string
//cs.MustRead("prefix.key", val)


fmt.Println(val)

```
