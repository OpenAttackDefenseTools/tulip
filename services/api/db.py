#!/usr/bin/env python
# -*- coding: utf-8 -*-

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

import re

from bson import ObjectId
from pymongo import MongoClient
from pymongo.errors import ServerSelectionTimeoutError
import sys
import pprint
from configurations import mongo_server, services


class DB:
    def __init__(self):
        try:
            self.client = MongoClient(
                mongo_server, serverSelectionTimeoutMS=200, unicode_decode_error_handler='ignore')
            self.client.server_info()
            self.db = self.client.pcap
            self.pcap_coll = self.db.pcap
            self.file_coll = self.db.filesImported
            self.signature_coll = self.db.signatures
            self.tag_col = self.db.tags

        except ServerSelectionTimeoutError as err:
            sys.stderr.write("MongoDB server not active on %s\n%s" %
                             (mongo_server, err))
            sys.exit(1)

    def getFlowList(self, filters):
        f = {}
        if "flow.data" in filters:
            f["flow.data"] = re.compile(filters["flow.data"], re.IGNORECASE)
        if "dst_ip" in filters:
            f["dst_ip"] = filters["dst_ip"]
        if "dst_port" in filters:
            if int(filters["dst_port"]) == -1:
                # remove dst_ip
                f.pop('dst_ip', None)
                f["dst_port"] = {
                    "$nin": [service["port"] for service in services]
                }
            else:
                f["dst_port"] = int(filters["dst_port"])
                
        if "from_time" in filters and "to_time" in filters:
            f["time"] = {"$gte": int(filters["from_time"]),
                         "$lt": int(filters["to_time"])}
        
        tag_queries = {}
        if "includeTags" in filters:
            tag_queries["$all"] = [str(elem) for elem in filters["includeTags"]]
        if "excludeTags" in filters:
            tag_queries["$nin"] = [str(elem) for elem in filters["excludeTags"]]

        if len(tag_queries.keys()) > 0:
            f["tags"] = tag_queries

        print("query:")
        pprint.pprint(f)

        return self.pcap_coll.find(f, {"flow": 0}).sort("time", -1).limit(2000)

    def getTagList(self):
        a = [i["_id"] for i in self.tag_col.find()]
        return a

    def getSignature(self, id):
        return self.signature_coll.find_one({"_id": id})

    def getFlowDetail(self, id):
        ret = self.pcap_coll.find_one({"_id": ObjectId(id)})
        ret["signatures"] = []
        for sig_id in ret["suricata"]:
            tmp = self.signature_coll.find_one({"_id": ObjectId(sig_id)})
            if tmp:
                ret["signatures"].append(tmp)
        
        return ret

    def setStar(self, flow_id, star):
        if star:
            self.pcap_coll.find_one_and_update({"_id": ObjectId(flow_id)}, {
                                               "$push": {"tags": "starred"}})
        else:
            self.pcap_coll.find_one_and_update({"_id": ObjectId(flow_id)}, {
                                               "$pull": {"tags": "starred"}})

    def isFileAlreadyImported(self, file_name):
        return self.file_coll.find({"file_name": file_name}).count() != 0

    def setFileImported(self, file_name):
        return self.file_coll.insert({"file_name": file_name})

    def insertFlows(self, filename, flows):
        if self.isFileAlreadyImported(filename):
            print("file already present! not importing it!")
            return
        result = self.pcap_coll.insert_many(flows)
        print("result: ", result)
        # IMPORTANT! create index for each field in the table if not present before
        # col.create_index([("time", ASCENDING)])
        # col.create_index([('flow.data', 'text')])
        return result

    def delete_all_pcaps(self, filename):
        return self.pcap_coll.remove({})
