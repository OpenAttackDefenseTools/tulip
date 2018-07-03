#!/usr/bin/env python

# This file is part of Flower.
#
# Copyright ©2018 Nicolò Mazzucato
# Copyright ©2018 Antonio Groza
# Copyright ©2018 Brunello Simone
# Copyright ©2018 Alessio Marotta
# DO NOT ALTER OR REMOVE COPYRIGHT NOTICES OR THIS FILE HEADER.
# 
# Flower is free software: you can redistribute it and/or modify
# it under the terms of the GNU General Public License as published by
# the Free Software Foundation, either version 3 of the License, or
# (at your option) any later version.
# 
# Flower is distributed in the hope that it will be useful,
# but WITHOUT ANY WARRANTY; without even the implied warranty of
# MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
# GNU General Public License for more details.
#
# You should have received a copy of the GNU General Public License
# along with Flower.  If not, see <https://www.gnu.org/licenses/>.

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