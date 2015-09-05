# degdb
Distributed Economic Graph Database

[Design Doc/Ramble](https://docs.google.com/document/d/1Z1zUMOGzsBLOU1JoeY-CLFI9eSMajrnQraBvSybjP8I/edit)

## WTFQuery Language

Weird bastardization of an internal Google query language and my own stuff.

```go
Id("degdb:foo").Preds("/type/object/name")
Filter("/type/object/name" == "Barack Obama")
```
