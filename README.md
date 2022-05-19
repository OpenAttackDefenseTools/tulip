# Tulip

Tulip is a flow analyzer meant for use during Attack / Defence CTF competitions. It is originally based on [flower](https://github.com/secgroup/flower), but it contains quite some changes:
* New front-end (typescript /react / tailwind)
* New ingestor code, based on gopacket
* Added ability to correlate flows with suricata alerts


## Configuration
Before starting the stack, edit `services/configurations.py`:

```
vm_ip = "10.10.3.1"

services = [{"ip": vm_ip, "port": 9876, "name": "cc_market"},
            {"ip": vm_ip, "port": 80, "name": "maze"},
            {"ip": vm_ip, "port": 8080, "name": "scadent"},
            {"ip": vm_ip, "port": 5000, "name": "starchaser"},
            {"ip": vm_ip, "port": 1883, "name": "scadnet_bin"}]
```


## Usage

The stack can be started with docker-compose:
```
docker-compose up -d --build
```
To ingest traffic, it is recommended to create a shared bind mount with the docker-compose. One convenient way to set this up is as follows:
1. On the vulnbox, start a rotating packet sniffer (e.g. tcpdump, suricata, ...)
1. Using rsync, copy complete captures to the machine running tulip (e.g. to /traffic)
1. Add a bind to flower-importer so it can read /traffic

The ingestor will use inotify to watch for new pcap's and suricata logs. No need to set a chron job.


## Suricata synchronization

Suricata alerts are read directly from the `eve.json` file. Because this file can get quite verbose when all extensions are enabled, it is recommended to strip the config down a fair bit. For example:
```yaml
# ...
  - eve-log:
      enabled: yes
      filetype: regular #regular|syslog|unix_dgram|unix_stream|redis
      filename: eve.json
      pcap-file: false
      community-id: false
      community-id-seed: 0
      types:
        - alert:
            metadata: yes
            # Enable the logging of tagged packets for rules using the
            # "tag" keyword.
            tagged-packets: yes
# ...
```

Sessions with matched alerts will be highlighted in the front-end and include which rule was matched.