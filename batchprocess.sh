#!/bin/bash

# Looks like
# Genesis 1 '/passage/?search=Genesis+1&version=ESV' > Genesis-1-ESV.tsv

# Get tsv file
python getchapters.py > biblechapters.tsv

# Update 'ESV' with translation when processing
cat biblechapters.tsv | awk '{print "echo Fetching ", $1,$2, " && python process_chapter.py",$1,$2,"'"'"'"$3"'"'"'",">",$1"-"$2"-ESV.tsv" }' | bash
