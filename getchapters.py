import requests
import sys
import re
from pyquery import PyQuery as pq

urls = dict()
urls['ESV'] = "https://www.biblegateway.com/versions/English-Standard-Version-ESV-Bible/#booklist"
urls['NIV'] = "https://www.biblegateway.com/versions/New-International-Version-NIV-Bible/#booklist"
urls['KJV'] = "https://www.biblegateway.com/versions/King-James-Version-KJV-Bible/#booklist"
urls['NLT'] = "https://www.biblegateway.com/versions/New-Living-Translation-NLT-Bible/#booklist"
urls['HCSB'] = "https://www.biblegateway.com/versions/Holman-Christian-Standard-Bible-HCSB/#booklist"


def get_chapters(translation):
    booklist = requests.get(urls[translation])
    bklist = pq(booklist.content)

    for ch in bklist("table.chapterlinks tr td.chapters a"):
        m = re.match('(\d*[a-zA-Z ]+) (\d+)', ch.attrib['title'])
        if m:
            book, chapter = m.groups()[0], m.groups()[1]
        else:
            raise ValueError("Cannot parse possibly malformed 'title' attribute: %s" % (ch.attrib['title']))

        yield book, chapter, ch.attrib['href']



if __name__ == "__main__":

    assert len(sys.argv) > 1, "Translation parameter is required"

    for ch in get_chapters(sys.argv[1]):
        print("%s\t%s\t%s" % ch)
