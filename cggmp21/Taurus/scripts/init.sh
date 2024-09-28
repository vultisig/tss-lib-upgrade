#!/bin/bash

declare -A droplets=(
    ["159.65.210.93"]="benchmark-london-01"
    ["159.65.209.59"]="benchmark-london-02"
    ["206.189.244.218"]="benchmark-london-03"
    ["143.198.243.26"]="benchmark-london-04"
    ["159.65.212.197"]="benchmark-london-05"
    ["46.101.67.139"]="benchmark-london-06"
    ["143.244.203.153"]="benchmark-nyc-01"
    ["146.190.197.105"]="benchmark-nyc-02"
    ["157.230.202.149"]="benchmark-nyc-03"
    ["157.230.67.109"]="benchmark-nyc-04"
    ["24.199.67.42"]="benchmark-nyc-05"
    ["24.144.66.162"]="benchmark-nyc-06"
    ["137.184.243.100"]="benchmark-nyc-07"
    ["146.190.202.249"]="benchmark-singapore-01"
    ["188.166.205.11"]="benchmark-singapore-02"
    ["139.59.220.155"]="benchmark-singapore-03"
    ["137.184.249.252"]="benchmark-singapore-04"
    ["139.59.217.237"]="benchmark-singapore-05"
    ["188.166.198.149"]="benchmark-singapore-06"
    ["139.59.219.80"]="benchmark-singapore-07"
)

for ip in "${!droplets[@]}"; do
    echo "Installing dependencies on ${droplets[$ip]} ($ip)"
    ssh root@$ip 'sudo apt update && sudo apt install -y golang-go'
done

echo "Dependencies installed on all droplets."