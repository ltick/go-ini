INI [![Build Status](https://travis-ci.org/ltick/go-ini.svg?branch=master)](https://travis-ci.org/ltick/go-ini) [![Sourcegraph](https://sourcegraph.com/github.com/ltick/go-ini/-/badge.svg)](https://sourcegraph.com/github.com/ltick/go-ini?badge)
===

Package go-ini provides INI file read and write functionality in Go.

[简体中文](README_ZH.md)

## Feature

- [x] Load multiple data sources(`[]byte`, file and `io.ReadCloser`) with overwrites.
- [x] Convenient usage of **unmarshal** like `json.Unmarshal` and `yaml.Unmarshal`.
- [x] Support **section** to classify key-value items.
- [x] Support **extend** to inherit key-value items from previous section.
- [x] Read with recursion values.
- [x] Read and auto-convert values to Go types.
- [x] Manipulate sections, keys and comments with ease.
- [ ] Read and **WRITE** comments of sections and keys.
- [ ] Read with multiple-line values.

## Installation

To use with latest changes:

	go get github.com/ltick/go-ini

Please add `-u` flag to update in the future.

## Getting Started


### Loading from data sources
