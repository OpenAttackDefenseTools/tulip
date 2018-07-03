#!/bin/sh

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

#questo script verrà eseguito sul server

echo "setup from ws!"
mkdir pcap
cd pcap
mkdir done

apt -y install apparmor-utils 
aa-complain /usr/sbin/tcpdump
echo "killing tcpdump"
pkill -f tcpdump
echo "executing tcpdump"
tcpdump -G 60  -w dump-%Y-%m-%d_%H:%M:%S.pcap -z "/root/pcap/delete_old_and_move.py"  port 80 or 5000 or 8080 or 9876

