#!/bin/bash

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

vm_host="10.10.3.1"

echo "setup server $vm_host"
echo "coping delete_old_and_move"
scp delete_old_and_move.py root@$vm_host:/root/pcap/delete_old_and_move.py  
echo "executing setup"
ssh root@$vm_host 'bash -s' < ./setup_ws.sh

echo "---> SETUP DONE!"

#./ws_pcap_importer.sh