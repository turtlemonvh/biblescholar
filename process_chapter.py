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
$ python process_chapter.py 'Genesis' '1' '1' '/passage/?search=Genesis+1&version=ESV' > gen1.tsv

Some verses end up like this:

Genesis 2       4        These are the generations
Genesis 2       4       of the heavens and the earth when they were created,
Genesis 2       4       in the day that the Lord God made the earth and the heavens.

Doesn't handle multi-downloads; just goes through each version for now

"""

parser = argparse.ArgumentParser(description='Process some integers.')
parser.add_argument('book', type=str, help='book of the Bible; e.g. Genesis')
parser.add_argument('chapter', type=int, help='chapter of the book; e.g. 1')
parser.add_argument('url', type=str, help='relative url to the page with content; e.g. /passage/?search=Genesis+1&version=ESV')
args = parser.parse_args()

url = 'https://www.biblegateway.com%s' % (args.url)
version = urlparse.parse_qs(url)['version'][0]

#print(url)
#sys.exit(0)

page = requests.get(url)
verses = pq(page.content)

# http://pyquery.readthedocs.io/en/latest/api.html#pyquery.pyquery.PyQuery.remove
# sup.versenum, sup.crossreference, sup.footnote
verses("table.passage-cols td.passage-col div.passage-text p > span.text sup").remove()
verses("table.passage-cols td.passage-col div.passage-text p > span.text span.chapternum").remove()

vss = verses("table.passage-cols td.passage-col div.passage-text p > span.text")

prev_verse = 0

for vs in vss:
    txt = vs.text_content().strip()

    grp = re.search("[a-zA-z]+\-(\d+)\-(\d+)", vs.attrib['class'])
    chapter, verse = grp.group(1), grp.group(2)

    if prev_verse == verse:
        # check for continuation
        line = u'%s %s' % (line, txt)
        continue

    line = u'%s\t%s\t%s\t%s\t%s' % (version, args.book, args.chapter, verse, txt)
    line.encode('utf8')
    print(line)

    prev_verse = verse



