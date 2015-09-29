# degdb
[![Build Status](https://travis-ci.org/DegDB/degdb.svg?branch=master)](https://travis-ci.org/DegDB/degdb)
[![GoDoc](https://godoc.org/github.com/DegDB/degdb?status.svg)](https://godoc.org/github.com/DegDB/degdb)

Distributed Economic Graph Database

[Design Doc/Ramble](https://docs.google.com/document/d/1Z1zUMOGzsBLOU1JoeY-CLFI9eSMajrnQraBvSybjP8I/edit)

Initial implementation done at PennApps 2015 Fall. It can be located in the `old/` directory. There is a slow rewrite/redesign happening.

## Running
```bash
go run main.go -new -peers="example.com:8181,foo.io:8182"
```

## Development
For development purposes you can launch multiple nodes within a single binary. This can only be used in development and disables connecting to external peers.
```bash
go run main.go -new -port=8181 -nodes=10
```

## License

DegDB is licensed under the MIT license.
