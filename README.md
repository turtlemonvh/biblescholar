# Bible Search

An application for searching several versions of the Bible at the same time, esp. with Alexa.

See [this series of blog posts](http://turtlemonvh.github.io/tag/biblescholar.html) for information about the set up, including:

* scraping data
* working with bleve
* deploying a skill to AWS Elasticbeanstalk
* handling security requirements for Alexa apps (esp. SSL configuration)
* and more!

Deployed to

* Website: [https://www.biblescholarsearch.net/](https://www.biblescholarsearch.net/)
* Amazon Alexa App Store: [https://www.amazon.com/dp/B01N4JOMQ3/](https://www.amazon.com/dp/B01N4JOMQ3/)

## Structure

See the `scrape` folder for python scripts to download TSVs of Bible verses.

See the `search` folder for a golang app to index and search this data.

## Grabbing data

This will create a TSV called "ESV.tsv" with the entire contents of this translation in a single TSV file.

```bash
cd scrape
export TRANSLATION=ESV
python batchprocess.py
```

## Running the server

```bash
cd search

# Build for mac
make darwin

# Run the web server
./artifacts/biblescholar-darwin-amd64 server -p 8080
```

## TODO

* Check the `search` README for more TODOs

