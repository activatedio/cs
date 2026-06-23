> ## CS
>
> This is a new library aiming to simplify runtime configuration for Go
> applications. Expect more updates shortly.
>


[![GitHub Workflow Status](https://img.shields.io/github/actions/workflow/status/activatedio/cs/ci.yaml?branch=main&style=flat-square)](https://github.com/activatedio/cs/actions?query=workflow%3ACI)
[![Go Report Card](https://goreportcard.com/badge/github.com/activatedio/cs?style=flat-square)](https://goreportcard.com/report/github.com/activatedio/cs)

# CS

CS - Config Source - Runtime library for flexible configuration.

## Install

``` sh
go get -u github.com/activatedio/cs

```

## Usage

The following example shows how to create a new config, add sources, and retrieve values

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

## Locked sources

`AddLockedSource` registers a source whose values are **locked**: once applied,
no other source — and no late-binding (env) source — may override any key it
provides, nor any descendant of it ("from that point of the graph down").
Locked values bypass the usual zero-value skip, so a lock to `false` / `0` /
`""` is honored, and they render with a `[locked]` marker in `Dump`. This is the
building block for shipping hardened, compile-time-pinned configuration.

``` go
cfg := cs.New()

cfg.AddSource(cs.FromValue("session", &SessionConfig{DisableSecure: true}))
cfg.AddLateBindingSource(cs.FromEnvironmentVars()) // SESSION_DISABLE_SECURE=true

// Pin the value regardless of file or env.
cfg.AddLockedSource(cs.FromValue("session.disableSecure", false))

var disableSecure bool
cfg.MustRead("session.disableSecure", &disableSecure) // always false
```
