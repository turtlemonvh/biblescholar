import requests
from pyquery import PyQuery as pq

urls = dict()
urls['ESV'] = "https://www.biblegateway.com/versions/English-Standard-Version-ESV-Bible/#booklist"
urls['NIV'] = "https://www.biblegateway.com/versions/New-International-Version-NIV-Bible/#booklist"
urls['KJV'] = "https://www.biblegateway.com/versions/King-James-Version-KJV-Bible/#booklist"
urls['NLT'] = "https://www.biblegateway.com/versions/New-Living-Translation-NLT-Bible/#booklist"
urls['HCSB'] = "https://www.biblegateway.com/versions/Holman-Christian-Standard-Bible-HCSB/#booklist"

booklist = requests.get(urls['ESV'])
bklist = pq(booklist.content)
for ch in bklist("table.chapterlinks tr td.chapters a"):
    print("%s\t%s" % (ch.attrib['title'], ch.attrib['href']))

"""
As tuples:
items = [(ch.attrib['title'], ch.attrib['href']) for ch in bklist("table.chapterlinks tr td
.chapters a") ]

"""

