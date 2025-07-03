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
