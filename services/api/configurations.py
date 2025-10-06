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

import os
import json
from pathlib import Path

traffic_dir = Path(os.getenv("TULIP_TRAFFIC_DIR", "/traffic"))
dump_pcaps_dir = Path(os.getenv("DUMP_PCAPS", "/traffic"))
tick_length = os.getenv("TICK_LENGTH", 2*60*1000)
flag_lifetime = os.getenv("FLAG_LIFETIME", 5)
start_date = os.getenv("TICK_START", "2018-06-27T13:00:00+02:00")
flag_regex = os.getenv("FLAG_REGEX", "[A-Z0-9]{31}=")
vm_ip = os.getenv("VM_IP", "10.10.3.1")
visualizer_url = os.getenv("VISUALIZER_URL", "http://127.0.0.1:1337")

if os.getenv("SERVICES"):
    services = json.loads(os.environ["SERVICES"])
else:
    vm_ip_1 = "10.60.2.1"
    helper = '''
10.61.5.1:1237 CyberUni 4
10.61.5.1:1236 CyberUni 3
10.61.5.1:1235 CyberUni 1
10.61.5.1:1234 CyberUni 2
10.60.5.1:3003 ClosedSea 1
10.60.5.1:3004 ClosedSea 2
10.62.5.1:5000 Trademark
10.63.5.1:1337 RPN
'''
    services = [
        {
            "ip": x.split(" ")[0].split(":")[0],
            "port": int(x.split(" ")[0].split(":")[1]),
            "name": " ".join(x.split(" ")[1:])
        }
        for x in helper.strip().split("\n")
    ]
    services += [{"ip": vm_ip_1, "port": -1, "name": "other"}]
