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
# I assume only services/test_pcap.pacp has been imported

import requests
import base64
import binascii

#WS_URL = "http://localhost:5000"
#FE_URL = "http://localhost:3000"

WS_URL = "http://flower-python:5000"
FE_URL = "http://flower-node:3000"

def get_first_flow_id():
    res = requests.post("{}/query".format(WS_URL), json={}).json()
    return res[0]["_id"]["$oid"]

FLOW_ID = get_first_flow_id()

def do_request(path):
    return requests.get("{}/{}".format(WS_URL,path))

def test_services():
    services = do_request("services").json()
    assert len(services) == 5
    assert services[0]["ip"] == "10.10.3.1"

def test_query():
    res = requests.post("{}/query".format(WS_URL), json={}).json()
    assert len(res) == 539

def test_frontend():
    assert "You need to enable JavaScript to run this app." in requests.get("{}".format(FE_URL)).text
    # todo find a better way to test this, maybe

def test_flow():
    flow = requests.get("{}/flow/{}".format(WS_URL,FLOW_ID)).json()
    assert len(flow["flow"]) == 70
    # non-printable char are replaced with other things, so we check only the first
    for p in flow["flow"][:1]:
        assert binascii.hexlify(p['data'].encode('ascii')).decode('ascii') == p["hex"]

    assert flow["src_port"] == 38910
    assert flow["dst_port"] == 9876
    assert flow["src_ip"] == "10.10.3.126"
    assert flow["dst_ip"] == "10.10.3.1"
    assert flow["time"] == 1530098790268
    assert flow["duration"] == 457

def test_convert_to_request():
    # todo
    pass

def test_convert_to_pwntools():
    # todo
    pass

def main():
    test_services()
    test_query()
    test_star()
    test_frontend()
    test_flow()

if __name__ == "__main__":
    main()
