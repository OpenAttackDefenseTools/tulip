
[![circleci][circleci-shield]][circleci-shield]
[![Contributors][contributors-shield]][contributors-url]
[![Forks][forks-shield]][forks-url]
[![Pull requests][pr-shield]][pr-url]
[![Stargazers][stars-shield]][stars-url]
[![Issues][issues-shield]][issues-url]
[![GPL License][license-shield]][license-url]

<!-- PROJECT LOGO -->
<br />
<p align="center">
  <a href="https://github.com/secgroup/flower">
    <img src="demo_images/cherry-blossom.png" alt="Logo" width="80" height="80">
  </a>
  <h3 align="center">Flower</h3>
  <p align="center">
    TCP flow analyzer with sugar for Attack/Defence CTF
    <br />
    <a href="https://github.coms/ecgroup/flower/issues">Report Bug</a>
    ·
    <a href="https://github.com/secgroup/flower/issues">Request Feature</a>
    ·
    <a href="#features">View Features</a>
  </p>
</p>

## Table of Contents

- [Table of Contents](#table-of-contents)
- [What is it?](#what-is-it)
- [Features](#features)
- [Getting Started](#getting-started)
  - [Run with docker](#run-with-docker)
  - [Manual installation](#manual-installation)
    - [Run](#run)
  - [Pcap import](#pcap-import)
- [Security tips (Important!)](#security-tips-important)
- [Credits](#credits)

## What is it?

![demo_image](https://github.com/secgroup/flower/blob/master/demo_images/demo3.png?raw=true)

Flower is an automatic packet analyzer made by Ca' Foscari University team for CyberChallenge attack/defense CTF held in Rome on the June 27th, 2018.

This tool was written in less than ten days, but it works! Every **contribution** is welcome!

Presentation of Flower (from min 7:30), and general introduction to CTFs at ESC2K18 in italian:

[![tools presentation](http://img.youtube.com/vi/oGB7LFwTghE/0.jpg)](http://www.youtube.com/watch?v=oGB7LFwTghE)

## Features
- Only one command needed to have it up, thanks to docker.
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
- Automatic export GET/POST requests directly in python format
![](https://github.com/secgroup/flower/blob/master/demo_images/demo_request_export.png)
- Automatic export to pwntools
![](https://github.com/secgroup/flower/blob/master/demo_images/demp_export_pwn.png)

## Getting Started

### Run with docker

Clone the repo, enter in the directory, and just run `docker-compose up`, and after a while you will find flower at [http://localhost:3000](http://localhost:3000).

For the flag regex, modify `REACT_APP_FLAG_REGEX` in `docker-compose.yml`.

The build will automatically import the test pcaps.

To enter in the service to import other pcaps, run `docker exec -it flower_flower-python_1 /bin/bash` (if flower is in a folder with a different name, modify the prefix after `-it`).
The container share the `/shared` folder with the host. Put the pcap files inside this folder and use `python services/importer.py /shared/pcap_file_here` from the container to import pcaps to flower.

### Manual installation

1. Clone and install dependencies
    ```bash
    git clone https://github.com/secgroup/flower
    cd flower
    npm install 
    pip install -r services/requirements.txt
    ```
2. (Optional) Set the following environment variables:
- `REACT_APP_FLOWER_MONGO` ip of the host that will have flower db active (mongodb)
- `REACT_APP_FLOWER_SERVICES` ip of the host that will have services active
- `REACT_APP_FLAG_REGEX` regex that match flags. 

3. Mongodb is required on the same machine that run the services.
To start it: `sudo mongod --dbpath /path/to/mongodb/db --bind_ip 0.0.0.0` 


#### Run
1. Start flower
    ```bash
    ./run.sh
    ```
2. Start flower services
    ```bash
    cd services
    ./run_ws.sh
    ```
Once everything has been started, flower should be accessible at the address of the machine that started it on port 3000.


### Pcap import
You must first install pynids from [here](https://github.com/MITRECND/pynids). The pip version is outdated! Good luck with the installation.
Then, you can import pcaps into mongodb by executing the provided script `importer.py` as follows:
```
cd services
./importer.py pcap_file.pcap
```
You can find a test_pcap in `services/test_pcap`. For a quick demo, run `./importer.py test_pcap/dump-2018-06-27_13:25:31.pcap`

## Security tips (Important!)

If you are going to use flower in a CTF, remember to set up the firewall in the most appropriate way, as the current implementation does not use other security techniques.
> If you ignore this, everybody will be able to connect to your database and steal all your flags!


## Credits
- [Nicolò Mazzucato](https://github.com/nicomazz)
- Antonio Groza
- Simone Brunello
- Alessio Marotta

With the support of [c00kies@venice](https://secgroup.github.io/)


<!-- MARKDOWN LINKS & IMAGES -->
<!-- https://www.markdownguide.org/basic-syntax/#reference-style-links -->
[circleci-shield]: https://circleci.com/gh/secgroup/flower.svg?style=shield

[contributors-shield]: https://img.shields.io/github/contributors/secgroup/flower.svg?style=flat-square
[contributors-url]: https://github.com/secgroup/flower/graphs/contributors

[forks-shield]: https://img.shields.io/github/forks/secgroup/flower.svg?style=flat-square
[forks-url]: https://github.com/secgroup/flower/network/members

[stars-shield]: https://img.shields.io/github/stars/secgroup/flower.svg?style=flat-square
[stars-url]: https://github.com/secgroup/flower/stargazers

[issues-shield]: https://img.shields.io/github/issues/secgroup/flower.svg?style=flat-square
[issues-url]: https://github.com/secgroup/flower/issues

[license-shield]: https://img.shields.io/github/license/secgroup/flower.svg?style=flat-square
[license-url]: https://github.com/secgroup/flower/blob/master/LICENSE.txt

[pr-shield]: https://img.shields.io/github/issues-pr/secgroup/flower.svg?style=flat-square
[pr-url]: https://github.com/secgroup/flower/pulls

[product-screenshot]: images/screenshot.png
