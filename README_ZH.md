INI [![Build Status](https://travis-ci.org/ltick/go-ini.svg?branch=master)](https://travis-ci.org/ltick/go-ini) [![Sourcegraph](https://sourcegraph.com/github.com/ltick/go-ini/-/badge.svg)](https://sourcegraph.com/github.com/ltick/go-ini?badge)
===

Package go-ini provides INI file read and write functionality in Go.

[English](README.md)

## 特性

- [x] 灵活的数据源(`[]byte`, 文件 及 `io.ReadCloser`).
- [x] 提供 **unmarshal** 接口方法 `json.Unmarshal` and `yaml.Unmarshal`.
- [x] 支持 **section**(切片) 对键值分类.
- [x] 支持 **extend**(扩展) 继承前一切片的所有键值.
- [x] 递归读取数据.
- [x] 将值自动转换为指定的 Go 语言原生类型.
- [x] 便捷操作 sections, keys 以及 comments.
- [ ] Read and **WRITE** comments of sections and keys.
- [ ] Read with multiple-line values.

## 安装

获取最新代码:

	go get github.com/ltick/go-ini

希望保持更新请使用`-u`参数:

	go get -u github.com/ltick/go-ini

## Getting Started

### Loading from data sources
