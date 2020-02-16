# Bible Search

A go application using bleve to index tsvs of Bible verses. Uses go modules and go 1.13+.

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
./artifacts/biblescholar-darwin-amd64 index -d ../scrape
```

### Text search

```bash
# Use the built in bleve search tool to query the index
# After "go install github.com/blevesearch/bleve/cmd/bleve"
bleve query verses.bleve "for God so loved the world"
```

### Running server

```bash
# Start server
./artifacts/biblescholar-darwin-amd64 server -i verses.bleve

# Example alexa request
cat test/exampleAlexaRequest.json | jq .

# Example alexa response
curl -s -X POST localhost:8000/alexa/search -d '@test/exampleAlexaRequest.json' | jq .
```

## Working with ELB

### Basic tooling

```
# Install tools
pip install awscli
pip install --upgrade awsebcli

# Add a file that exports env variables
# Setting up the ~/.aws/credentials and ~/.aws/config files didn't work for me for `eb` even though they worked for `aws` commands
vi .aws

# Set this as the active profile
export AWS_DEFAULT_PROFILE=biblescholar

# Initialize
eb init
```

### Dockerizing

I did local testing with [docker-machine](https://docs.docker.com/machine).

```bash
# Build
docker build -t biblescholar .

# Run docker container
# DO NOT try to run with the index mounted as a volume. Bleve will panic when it tries to work with a fs mutex. :(
docker run -it -d --name biblescholar -p 8000:80 biblescholar

# Test request
curl localhost:8000

# Try a more complex request
curl -s -X POST localhost:8000/alexa/search -d '@test/exampleAlexaRequest.json'
```

### Domain name

Running on amazon with urls

* https://biblescholarsearch.net
* http://biblescholar-env.us-west-2.elasticbeanstalk.com

```
# Alexa url will reject your request due to required cert validation in production
curl -s -X POST "$url/alexa/search" -d '@test/exampleAlexaRequest.json' | jq .

# This works though
curl -X GET "$url/search?q=cats%20dogs&size=2&highlight" | jq .
```

## TODO

* Work with richer struct types instead of raw gabs objects for request handling
* Try out running docker locally this way
    * https://docs.aws.amazon.com/elasticbeanstalk/latest/dg/create_deploy_docker-eblocal.html
* More ebextensions
    * http://docs.aws.amazon.com/elasticbeanstalk/latest/dg/customize-containers-ec2.html
* see FIXMEs in code too
    * separate card response from voice response
* pass ctx through request chain into search object, use to trace log items together, handle request termination
* better response formatting


### Nice to haves

* look into streaming back a recording of a verse (from S3) instead of having alexa read it
* read a section of scripture

    ```
    // optionally "from the {translation} translation"
    range
    - read {book} {chapter} {verse} to {verse}
    - read {book} chapter {chapter} verse {verse} to verse {verse}
    single
    - read {book} {chapter} {verse}
    - read {book} chapter {chapter} verse {verse}
    chapter
    - read {book} {chapter}
    - read {book} chapter {chapter}
    ```

