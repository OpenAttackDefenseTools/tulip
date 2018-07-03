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

#convert a flow into pwn script
def flow2pwn(flow):
    ip = flow["dst_ip"]
    port = flow["dst_port"]

    script = """from pwn import *

proc = remote('{}', {})
""".format(ip, port)

    for message in flow['flow']:
        if message['from'] == 's':
            script += """proc.writeline("{}")\n""".format(message['data'][:-1])

        else:
            for m in range(len(message['data'])):
                script += """proc.recvuntil("{}")\n""".format(message['data'][-20:].replace("\n","\\n"))
                break

    return script
