# Bible Search

An application for searching several versions of the Bible at the same time.  The plan is to hook set this up as an Alexa skill before the end of the year.

## Structure

See the `scrape` folder for python scripts to download TSVs of Bible verses.

See the `search` folder for a golang app to index and search this data.

## Grabbing data

This will create a TSV called "ESV.tsv" with the entire contents of this translation in a single TSV file.

```bash
# Inside the 'scrape' directory
export TRANSLATION=ESV
python batchprocess.py

```

## TODO

* Check the `search` README for more TODOs
* Check for errors in scraping utilities; audit by looking for duplicated or missing verses (dict diff)

```
# This is suspicious...
$ wc -l *.tsv
   30926 HCSB.tsv
   31102 KJV.tsv
   30852 NIV.tsv
   92880 total
```

