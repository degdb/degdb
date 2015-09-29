# degdb
[![Build Status](https://travis-ci.org/DegDB/degdb.svg?branch=master)](https://travis-ci.org/DegDB/degdb)
[![GoDoc](https://godoc.org/github.com/DegDB/degdb?status.svg)](https://godoc.org/github.com/DegDB/degdb)

Distributed Economic Graph Database

[Design Doc/Ramble](https://docs.google.com/document/d/1Z1zUMOGzsBLOU1JoeY-CLFI9eSMajrnQraBvSybjP8I/edit)

Initial implementation done at PennApps 2015 Fall.

There is a slow rewrite/redesign happening.

## WTFQuery Language

This is a weird combination of Go syntax and other stuff. Due to efficiency, it's parsed using the Go parser library. Future changes include supporting Gremlin and other actual graph languages.

```go
// Fetch by Id and then get the name
Id("degdb:foo").Preds("/type/object/name")

// Find all topics with name "Barack Obama"
Filter("/type/object/name" == "Barack Obama")

// Find all nodes
All()
```

## License

DegDB is licensed under the MIT license.
