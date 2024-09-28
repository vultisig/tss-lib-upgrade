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
THRESHOLD=13

run_on_droplet() {
    local ip=$1
    local id=${droplets[$ip]}
    
    local cleanup_command="pkill main; pkill go; sleep 2; fuser -k 54321/tcp"
    local ssh_command="cd ~/Taurus && go run main.go $THRESHOLD $id 0.0.0.0:54321"
    
    # Add all droplets to the command
    for droplet_ip in "${!droplets[@]}"; do
        local droplet_id=${droplets[$droplet_ip]}
        ssh_command+=" $droplet_id:$droplet_ip:54321"
    done
    
    echo "Running on $id ($ip):"
    
    expect_script=$(cat <<EOF
spawn ssh root@$ip
expect "# "
send "$cleanup_command\r"
expect "# "
send "$ssh_command\r"
expect "Type 'start' when everyone is connected:"
sleep 10
send "start\r"
interact
EOF
)
    
    gnome-terminal -- expect -c "$expect_script"
}

for ip in "${!droplets[@]}"; do
    run_on_droplet $ip
done

echo "All commands have been initiated in separate terminals. 'start' commands will be sent after 10 seconds in each terminal."