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

import base64

def escape(i):
    if isinstance(i, str):
        i = ord(i)
    ret = chr(i) if 0x20 <= i and i < 0x7f else f'\\x{i:02x}'
    if ret in '\\"':
        ret = '\\' + ret
    return ret

def convert(message):
    return ''.join([escape(i) for i in message])

#convert a flow into pwn script
def flow2pwn(flow):
    ip = flow["dst_ip"]
    port = flow["dst_port"]

    script = """from pwn import *
import sys

host = sys.argv[1]
proc = remote(host, {})
""".format(port)

    for message in flow['flow']:
        data = base64.b64decode(message["b64"])
        if message['from'] == 'c':
            script += """proc.write(b"{}")\n""".format(convert(data))

        else:
            script += """proc.recvuntil(b"{}")\n""".format(convert(data[-10:]).replace("\n","\\n"))

    return script

