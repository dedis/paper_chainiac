#!/usr/bin/env python

from bs4 import BeautifulSoup
import urllib.request
import os
import re

date = 201505
to = 201702
snap = "http://snapshot.debian.org/archive/debian/"

while date < to:
    year = int(date / 100)
    month = date % 100
    print("Current date: %d - %d - %d" % (date, year, month))

    url = snap + "?year=%d&month=%02d" % (year, month)
    r = urllib.request.urlopen(url).read()
    soup = BeautifulSoup(r, "lxml")
    for d in soup.find_all('a'):
        h = d.get("href")
        if h.startswith(str(year)):
            url = snap + h + "dists/stretch/main/binary-amd64/"
            for f in ["Release", "Packages.gz"]:
                save = re.compile('[TZ/]').sub('', h) + "_" + f
                if not os.path.exists(save):
                    print(url + f, save)
                    urllib.request.urlretrieve(url + f, save)

    month += 1
    if month == 13:
        month = 1
        year += 1
    date = year * 100 + month
