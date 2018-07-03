#!/bin/bash

gnome-terminal --  pwd; sleep 10 && 
gnome-terminal --  ctft;pwd; sleep 200;

#gnome-terminal --tab -e "sudo mongod --dbpath /mnt/DATA/mongodb --bind_ip 0.0.0.0" --tab -e "ctft; cd flower;npm start;" --tab -e "ctft; cd flower-services; ./run_ws.sh"

