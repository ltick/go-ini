INI [![Build Status](https://travis-ci.org/ltick/go-ini.svg?branch=master)](https://travis-ci.org/ltick/go-ini) [![Sourcegraph](https://sourcegraph.com/github.com/ltick/go-ini/-/badge.svg)](https://sourcegraph.com/github.com/ltick/go-ini?badge)
===

Package go-ini provides INI file read and write functionality in Go.

[简体中文](README_ZH.md)

## Feature

- Load multiple data sources(`[]byte`, file and `io.ReadCloser`) with overwrites.
- Convenient usage of **unmarshal** like `json.Unmarshal` and `yaml.Unmarshal`.
- Support **extend** key-value items from previous section.
- Read with recursion values.
- Read with multiple-line values.
- Read and auto-convert values to Go types.
- Read and WRITE comments of sections and keys.
- Manipulate sections, keys and comments with ease.

## Installation

To use with latest changes:

	go get github.com/ltick/go-ini

Please add `-u` flag to update in the future.

## Getting Started

### Loading from data sources
