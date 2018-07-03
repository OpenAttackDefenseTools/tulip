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
import os

ws_ip = os.getenv("REACT_APP_FLOWER_SERVER_IP", "0.0.0.0")
mongo_server = 'mongodb://' + ws_ip + ':27017/'
vm_ip = "10.10.3.1"  # todo put regex

services = [{"ip": vm_ip, "port": 9876, "name": "cc_market"},
            {"ip": vm_ip, "port": 80, "name": "maze"},
            {"ip": vm_ip, "port": 8080, "name": "scadent"},
            {"ip": vm_ip, "port": 5000, "name": "starchaser"},
            {"ip": vm_ip, "port": 1883, "name": "scadnet_bin"}]


def containsFlag(text):
    # todo implementare logica contains
    regex_flag = os.getenv("REACT_APP_FLAG_REGEX", r'[A-Z0-9]{31}=')
    match = re.match(regex_flag, text)
    return match
