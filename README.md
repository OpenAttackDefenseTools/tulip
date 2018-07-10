Flower
======
Automatic packet analyzer made by Ca' Foscari team (unive) for CyberChallenge attack/defense CTF of 27/06/2018.
This tool was written in less than ten days. Every **pull request** is welcome!

![](https://github.com/secgroup/flower/blob/master/demo_images/demo3.png?raw=true)

## Install
```bash
git clone https://github.com/secgroup/flower
cd flower
npm install 
pip install -r services/requirements.txt
```

## Setup
Env var to set:
- `REACT_APP_FLOWER_SERVER_IP` ip of the host that will have flower services and db active
- `REACT_APP_FLAG_REGEX` regex that match flags. 
Mongodb is required on the same machine that run the services.
To start it: `sudo mongod --dbpath /path/to/mongodb/db --bind_ip 0.0.0.0` 


## Run

#### Start flower
```bash
./run.sh
```
#### Start flower services
```bash
cd services
./run_ws.sh
```
Once everything has been started, flower should be accessible at the address of the machine that started it on port 3000.


## Pcap import
You must first install pynids from [here](https://github.com/MITRECND/pynids). The pip version is outdated! Good luck with the installation.
Then, you can import pcaps into mongodb by executing the provided script `importer.py` as follows:
```
cd services
./importer.py pcap_file.pcap
```
You can find a test_pcap in `services/test_pcap`. For a quick demo, run `./importer.py test_pcap/dump-2018-06-27_13:25:31.pcap`

## Security tips
If you are going to use flower in a CTF, remember to set up the firewall in the most appropriate way, as the current implementation does not use other security techniques.

## Features
- Flow list
- **Vim like navigation** ( `k` and `j` to navigate the list)
- Regex filtering with highlight
![](https://github.com/secgroup/flower/blob/master/demo_images/demo_search_hilight.png?raw=true)
- Highlight in red flow with flags
- Favourite management
- Time filter
- Service filter
![](https://github.com/secgroup/flower/blob/master/demo_images/demo_service_selection.png)
- Colored hexdump
![](https://github.com/secgroup/flower/blob/master/demo_images/demo_hex_dump.png?raw=true)
- Automatic export GET/POST request directly in python format
![](https://github.com/secgroup/flower/blob/master/demo_images/demo_request_export.png)
- Automatic export to pwntools
![](https://github.com/secgroup/flower/blob/master/demo_images/demp_export_pwn.png)

## Credits
- [Nicolò Mazzucato](https://github.com/nicomazz)
- Antonio Groza
- Simone Brunello
- Alessio Marotta

With the support of [c00kies@venice](https://secgroup.github.io/)
