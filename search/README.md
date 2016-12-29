# Bible Search

A go application using bleve to index tsvs of Bible verses.

## Basic usage

### Building

```bash
# Do everything
make

# Just build for mac
make darwin
```

### Indexing

```bash
# Run the index command using ESV tsv files
biblescholar index -d ../downloads/*-ESV.tsv
```

### Text search

```bash
# Use the built in bleve search tool to query the index
bleve query verses.bleve "for God so loved the world"
```

### Running server

```bash
# Start server
cd command
./biblescholar server -i ../../verses.bleve

# Example alexa request
cat ../test/exampleAlexaRequest.json | jq .

# Example alexa response
curl -s -X POST localhost:8000/alexa/search -d '@../test/exampleAlexaRequest.json' | jq .
```

## TODO

* Setup search in addition to indexing
* Add makefile including githash and branch in build
* vendor deps for commands
* rename binary: `go build -o biblescholar .`
* pass ctx through request chain into search object, use to trace log items together, handle request termination

## Nice to haves

* look into streaming back a recording of a verse (from S3) instead of having alexa read it

