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
artifacts/biblescholar-darwin-amd64 index -d ../downloads/*-ESV.tsv
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
artifacts/biblescholar-darwin-amd64 server -i ../../verses.bleve

# Example alexa request
cat ../test/exampleAlexaRequest.json | jq .

# Example alexa response
curl -s -X POST localhost:8000/alexa/search -d '@../test/exampleAlexaRequest.json' | jq .
```

## TODO

* vendor deps for commands
* pass ctx through request chain into search object, use to trace log items together, handle request termination

## Nice to haves

* look into streaming back a recording of a verse (from S3) instead of having alexa read it

