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

#sul server deve essere eseguito:
#cd ~/pcap_dumps
#tcpdump -G 120  -w dump-%Y-%m-%d_%H:%M:%S.pcap -z "./delete_old_and_move.py"  port 7789 or 5010 or 80 &
#dove vanno messe le porte dei servizi

importer_script_path='/mnt/DATA/cyberChallange/github/ctftools/flower-services/importer.py'
import_path='/mnt/DATA/pcap_dumps'

cd $import_path

while true 
do
    rsync -avzh root@10.0.1.1:~/pcap/done $import_path
    ultimo=$(find ./done -type f -printf '%T@ %p\n' | sort | tail -1 | cut -f2- -d" ")
    python $importer_script_path $ultimo
    sleep 60
done