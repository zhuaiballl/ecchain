#!/usr/bin/env bash
set -e

bootnode=`cat ./config/bootnode`
account=`cat ./config/account`

read -p "Output filename=?" filename

# kill server process gently
./kill_geth.sh

# run server at background
screen -Sdm pride_exp_geth ./bin/geth --experiment.output=./output/"$filename".txt --datadir ./gethdata --syncmode 'full' --port 30310 --bootnodes "$bootnode" --networkid 1114 --allow-insecure-unlock --http --http.addr 0.0.0.0 --http.port 8545 --http.corsdomain "*" --http.api "admin,eth,net,web3,miner,personal" --rpc.gascap 0  --rpc.txfeecap 0 --mine --miner.threads 1 --miner.etherbase "$account" --unlock "$account" --password ./password/password.txt
echo "Starting geth, sleep 5s..."
sleep 5
