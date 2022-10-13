#!/usr/bin/env python
# -*- coding: utf-8 -*-

import re

from bson import ObjectId
from pymongo import MongoClient
from pymongo.errors import ServerSelectionTimeoutError
import sys
import pprint

mongo_server = "localhost:27017"

client = MongoClient(mongo_server, serverSelectionTimeoutMS=200, unicode_decode_error_handler='ignore')
db = client.pcap
pcap_coll = db.pcap

pcap_coll.update_many(
    { },
    { "$pull": { "tags": { "$nin": [ "flag-in", "flag-out" ] }} }
)