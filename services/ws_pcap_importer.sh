#!/bin/sh

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