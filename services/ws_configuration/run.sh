#!/bin/bash

vm_host="10.10.3.1"

echo "setup server $vm_host"
echo "coping delete_old_and_move"
scp delete_old_and_move.py root@$vm_host:/root/pcap/delete_old_and_move.py  
echo "executing setup"
ssh root@$vm_host 'bash -s' < ./setup_ws.sh

echo "---> SETUP DONE!"

#./ws_pcap_importer.sh