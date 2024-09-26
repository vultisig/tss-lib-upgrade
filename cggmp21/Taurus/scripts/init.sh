#!/bin/bash

declare -A droplets=(
    ["178.128.171.139"]="benchmark-london-03"
    ["206.189.28.56"]="benchmark-london-01"
    ["178.128.169.97"]="benchmark-london-06"
    ["178.128.171.173"]="benchmark-london-05"
    ["178.128.169.240"]="benchmark-london-02"
    ["206.189.16.185"]="benchmark-london-04"
    ["157.245.152.187"]="benchmark-singapore-05"
    ["167.71.212.71"]="benchmark-singapore-06"
    ["178.128.121.19"]="benchmark-singapore-02"
    ["104.248.156.73"]="benchmark-singapore-01"
    ["157.230.243.21"]="benchmark-singapore-04"
    ["167.71.217.35"]="benchmark-singapore-07"
    ["167.71.211.145"]="benchmark-singapore-03"
    ["192.81.216.35"]="benchmark-nyc-05"
    ["167.172.128.89"]="benchmark-nyc-03"
    ["157.245.247.39"]="benchmark-nyc-01"
    ["167.172.141.133"]="benchmark-nyc-02"
    ["157.245.250.98"]="benchmark-nyc-04"
    ["167.172.129.33"]="benchmark-nyc-07"
    ["167.99.13.212"]="benchmark-nyc-06"
)

for ip in "${!droplets[@]}"; do
    echo "Installing dependencies on ${droplets[$ip]} ($ip)"
    ssh root@$ip 'sudo apt update && sudo apt install -y golang-go'
done

echo "Dependencies installed on all droplets."