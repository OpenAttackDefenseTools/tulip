#!/bin/bash
if [ -z "$TCPDUMP_CMD" ]; then
	echo "TCPDUMP_CMD not set in .env, exiting now assuming I'm not needed"
	exit 1
fi

while true
do
	$TCPDUMP_CMD | nc -l 11337
	echo "tcpdump command stopped, sleeping for 5s and restarting"
	sleep 5
done
