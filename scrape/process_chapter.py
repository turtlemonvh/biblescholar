import argparse
import requests
import re
import sys
import urlparse
from pyquery import PyQuery as pq

"""
This function takes a relative url and extracts all verses from that page as a tsv.

# See: http://stackoverflow.com/questions/5203105/printing-a-utf-8-encoded-string
$ export PYTHONIOENCODING=UTF-8
$ python process_chapter.py 'Genesis' '1' '/passage/?search=Genesis+1&version=ESV' > gen1.tsv

Doesn't handle multi-downloads; just goes through each version for now

"""

def process_chapter(book, chapter, rel_url):
    """Generator that yields lines from this book, chapter, url combination
    """
    url = 'https://www.biblegateway.com%s' % (rel_url)
    version = urlparse.parse_qs(url)['version'][0]

    page = requests.get(url)
    verses = pq(page.content)

    # http://pyquery.readthedocs.io/en/latest/api.html#pyquery.pyquery.PyQuery.remove
    # sup.versenum, sup.crossreference, sup.footnote
    vstxt = verses("table.passage-cols td.passage-col div.passage-text")
    vstxt.find("p span.text sup").remove()
    vstxt.find("table span.text sup").remove()
    vstxt.find("p span.text span.chapternum").remove()

    # Maybe only grabbing one...?
    vss = vstxt.find("p span.text, table span.text")

    prev_verse = 0
    prev_line = ""

    for vs in vss:
        txt = vs.text_content().strip()

        grp = re.search("[a-zA-z]+\-(\d+)\-(\d+)", vs.attrib['class'])
        chapter, verse = grp.group(1), grp.group(2)

        # DEBUG
        #print(chapter, verse, txt)

        if prev_verse == verse:
            # Continuation; append
            prev_line = u'%s %s' % (prev_line, txt)
            continue

        # Yield the line from the previous iteration
        if prev_line != "":
            yield prev_line

        # Form the new line
        line = u'%s\t%s\t%s\t%s\t%s' % (version, book, chapter, verse, txt)
        line.encode('utf8')

        # Update tracking of previous line
        prev_line = line
        prev_verse = verse

    # Last line
    yield prev_line


if __name__ == "__main__":

    parser = argparse.ArgumentParser(description='Process some integers.')
    parser.add_argument('book', type=str, help='book of the Bible; e.g. Genesis')
    parser.add_argument('chapter', type=int, help='chapter of the book; e.g. 1')
    parser.add_argument('url', type=str, help='relative url to the page with content; e.g. /passage/?search=Genesis+1&version=ESV')
    args = parser.parse_args()

    for line in process_chapter(args.book, args.chapter, args.url):
        print(line)

