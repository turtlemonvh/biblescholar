#!/bin/python

import codecs
import os
from getchapters import get_chapters
from process_chapter import process_chapter

TRANSLATION = os.environ.get("TRANSLATION", "NIV")

with codecs.open("%s.tsv" % (TRANSLATION), "w+", encoding='utf-8') as f:
    for book, chapter, rel_url in get_chapters(TRANSLATION):
        print("Grabbing verses for: %s %s" % (book, chapter))
        for line in process_chapter(book, chapter, rel_url):
            line.encode('utf8')
            f.write("%s\n" % line)

