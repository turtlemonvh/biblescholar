# Bible Search

A go application using bleve to index tsvs of Bible verses.

## Basic usage

```
# Set up build path
ln -s $(pwd) $GOPATH/src/github.com/turtlemonvh/bblsearch

# Go to buildpath and build
cd $GOPATH/src/github.com/turtlemonvh/bblsearch/command
go build .

# cd back to starting location
cd -

# Run with tsv files downloaded
# Run the search command using files in the "downloads" directory
command/command index downloads/


```

## TODO

* Setup search in addition to indexes
* Make indexing work per tsv instead of gathering all items at once

