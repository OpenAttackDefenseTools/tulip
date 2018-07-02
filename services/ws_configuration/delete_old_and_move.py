#!/usr/bin/env python

from sys import argv
from time import sleep
import os

COUNT = 120
DIRNAME = './done/'

os.chdir(DIRNAME)
files = os.listdir(".")

if len(files) > COUNT:
# assuming file name YYYY-MM-DD_HH:MM:SS.pcap
   oldest_file = sorted(files)[0]
   os.remove(oldest_file)

to_rename= '../' + argv[1]
print "to rename: ",to_rename

os.rename('../' + argv[1], argv[1])

#usare latin1 testare con urandom