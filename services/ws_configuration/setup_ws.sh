#!/bin/sh
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

