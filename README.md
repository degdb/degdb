# degdb [![Build Status](https://travis-ci.org/degdb/degdb.svg?branch=master)](https://travis-ci.org/degdb/degdb) [![Coverage Status](https://coveralls.io/repos/degdb/degdb/badge.svg?branch=master&service=github)](https://coveralls.io/github/degdb/degdb?branch=master) [![GoDoc](https://godoc.org/github.com/degdb/degdb?status.svg)](https://godoc.org/github.com/degdb/degdb)

Distributed Economic Graph Database

[Design Doc/Ramble](https://docs.google.com/document/d/1Z1zUMOGzsBLOU1JoeY-CLFI9eSMajrnQraBvSybjP8I/edit)

Initial implementation done at PennApps 2015 Fall. There is a slow rewrite/redesign happening.

## Running
```bash
# Install degdb and bitcoin dependencies
$ go get -v -u github.com/degdb/degdb github.com/btcsuite/btcwallet github.com/btcsuite/btcd

# Create the bitcoin wallet
$ btcwallet --create

# Launch new server and connect to provided peers.
$ go run main.go -peers="example.com:8181,foo.io:8182"
```

`$GOPATH/bin` must be on the path so degdb can launch instances of btcwallet and btcd.

## Development
For development purposes you can launch multiple nodes within a single binary. This can only be used in development and disables connecting to external peers.
```bash
go run main.go -port=8181 -nodes=10
```

## License

DegDB is licensed under the [MIT license](https://opensource.org/licenses/MIT).

## Contributors

* [Tristan Rice](https://fn.lc)
* [Chaoyi Zha](https://github.com/cydrobolt)
