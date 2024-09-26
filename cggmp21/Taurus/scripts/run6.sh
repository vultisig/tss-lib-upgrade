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
)

THRESHOLD=4

run_on_droplet() {
    local ip=$1
    local id=${droplets[$ip]}
    
    local cleanup_command="pkill main; pkill go; sleep 2; fuser -k 8080/tcp"
    local ssh_command="cd ~/Taurus && go run main.go $THRESHOLD $id $ip:8080"
    
    # Add all droplets to the command
    for droplet_ip in "${!droplets[@]}"; do
        local droplet_id=${droplets[$droplet_ip]}
        ssh_command+=" $droplet_id:$droplet_ip:8080"
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