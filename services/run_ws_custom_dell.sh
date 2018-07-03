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

gnome-terminal --  pwd; sleep 10 && 
gnome-terminal --  ctft;pwd; sleep 200;

#gnome-terminal --tab -e "sudo mongod --dbpath /mnt/DATA/mongodb --bind_ip 0.0.0.0" --tab -e "ctft; cd flower;npm start;" --tab -e "ctft; cd flower-services; ./run_ws.sh"

