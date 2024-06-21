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
from pathlib import Path
import json

traffic_dir = Path(os.getenv("TULIP_TRAFFIC_DIR", "/traffic"))
tick_length = os.getenv("TICK_LENGTH", 2*60*1000)
start_date = os.getenv("TICK_START", "2018-06-27T13:00+02:00")
mongo_host = os.getenv("TULIP_MONGO", "localhost:27017")
flag_regex = os.getenv("FLAG_REGEX", "[A-Z0-9]{31}=")
mongo_server = f'mongodb://{mongo_host}/'

vm_ip = os.getenv("VM_IP", "10.10.3.1")
services_path = os.getenv("TULIP_SERVICES_PATH", None)

if services_path is not None:
    with open(services_path, 'r') as f:
        services = json.load(f)
    for service in services:
        if "ip" not in service:
            service["ip"] = vm_ip
else:
    services = [
        {"ip": vm_ip, "port": 9876, "name": "cc_market"},
        {"ip": vm_ip, "port": 80, "name": "maze"},
        {"ip": vm_ip, "port": 8080, "name": "scadent"},
        {"ip": vm_ip, "port": 5000, "name": "starchaser"},
        {"ip": vm_ip, "port": 1883, "name": "scadnet_bin"},
        {"ip": vm_ip, "port": -1, "name": "other"}
    ]
