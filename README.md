# Flower

Automatic package analyzer made by Ca' Foscari team (unive) for CyberChallenge attack/defense CTF. 

![](https://github.com/cyberchallengeit-ve/ctftools/blob/master/flower/demo.png?raw=true)

Features:
- Flow list
- Regex filtering with hilight
- Hilight in red flow with flags
- Favourite management
- Time filter
- Service filter
- Colored hexdump

## Setup
env var to set:
- `REACT_APP_FLOWER_SERVER_IP` ip of the host that will have flower web services and db active
- `REACT_APP_FLAG_REGEX` regex that match flags

## Install
Install package dependencies:
```
npm install 
```
You must have mongodb installed and running in `REACT_APP_FLOWER_SERVER_IP` ip.


## Run
Start flower
```bash
services/run_ws.sh #better in a different window, to see what is happening
./run.sh
```
