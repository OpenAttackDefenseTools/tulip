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

traffic_dir = Path(os.getenv("TULIP_TRAFFIC_DIR", "/traffic"))
mongo_host = os.getenv("TULIP_MONGO", "0.0.0.0:27017")
mongo_server = f'mongodb://{mongo_host}/'
vm_ip = "192.168.201.2"  # todo put regex

services = [{"ip": vm_ip, "port": -1, "name": "unknown"},
            {"ip": vm_ip, "port": 8000, "name": "saarbahn"},
            {"ip": vm_ip, "port": 1984, "name": "bytewarden"},
            {"ip": vm_ip, "port": 5445, "name": "saarsecvv"},
            {"ip": vm_ip, "port": 8080, "name": "saarcloud"},
            {"ip": vm_ip, "port": 11025, "name": "saarloop"}]
