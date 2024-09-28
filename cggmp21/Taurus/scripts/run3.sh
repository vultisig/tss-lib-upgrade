#!/bin/bash

declare -A droplets=(
    ["159.65.210.93"]="benchmark-london-01"
    ["143.244.203.153"]="benchmark-nyc-01"
    ["146.190.202.249"]="benchmark-singapore-01"
)

THRESHOLD=2
PORT=54321

run_on_droplet() {
    local ip=$1
    local id=${droplets[$ip]}
    
    local cleanup_command="pkill main; pkill go; sleep 2; fuser -k $PORT/tcp"
    local ssh_command="cd ~/Taurus && go run main.go $THRESHOLD $id 0.0.0.0:$PORT"
    
    # Add all droplets to the command
    for droplet_ip in "${!droplets[@]}"; do
        local droplet_id=${droplets[$droplet_ip]}
        ssh_command+=" $droplet_id:$droplet_ip:$PORT"
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

echo "All commands have been initiated in separate terminals and 'start' commands will be sent after 10 seconds in each terminal."