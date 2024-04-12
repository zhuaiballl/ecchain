# Some Readme

## Step 0: [Both sides] Prepare files

Make sure the following commands are available.

```
bash python3 screen curl wget go gcc make tar jq
```

Compile and put the executables in the bin folder as follows.

```
# Put the executables as follows:
# cp [geth]./bin/geth
```

If for some reasons the x-permission is lost, grant the x-permission on executables.

```
chmod +x ./bin/*
chmod +x ./*.sh
chmod +x ./*.py
```

## Step 1: [Server side] Prepare accounts

The sealer account is already created. The address is placed at `./config/address`. The corresponding private key is placed at `./gethaccount/sealer/keystore/`.

However car accounts are not pre-created. Execute the following command to create 1000 car accounts.

```
echo 1000 | ./make_accounts.sh
```

## Step 1.5: [Server side] (if necessary) Edit genesis-template.json

```
{
  "config": {
    "chainId": 63898,
    "homesteadBlock": 0,
    "eip150Block": 0,
    "eip150Hash": "0x0000000000000000000000000000000000000000000000000000000000000000",
    "eip155Block": 0,
    "eip158Block": 0,
    "byzantiumBlock": 0,
    "constantinopleBlock": 0,
    "petersburgBlock": 0,
    "istanbulBlock": 0,
    "clique": {
      "period": 15, // time interval between blocks
      "epoch": 30000
    }
  },
  "nonce": "0x0",
  "timestamp": "0x62c0177c",
  "extraData": "0x0000000000000000000000000000000000000000000000000000000000000000b3270be37a758e67a67fc6f2b62247cc58e0e61f0000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
  "gasLimit": "0x01ffffffffffff",
  "difficulty": "0x1",
  "mixHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "coinbase": "0x0000000000000000000000000000000000000000",
  "alloc": {
    "b3270be37a758e67a67fc6f2b62247cc58e0e61f": {
      "balance": "0x200000000000000000000000000000000000000000000000000000000000000"
    }
  },
  "number": "0x0",
  "gasUsed": "0x0",
  "parentHash": "0x0000000000000000000000000000000000000000000000000000000000000000",
  "baseFeePerGas": null
```

## Step 2: [Server side] Fund accounts

The car accounts are not sealers, i.e. they can't mine (vote) to produce ether. The easiest way is to pre-fund them at the genesis block. It's advised to write a smart contract to fund new accounts for prodution use, but we omit it because it is just an experiment.

```
./make_genesis.py
```

## Step 3: [Server side] Initialize the blockchain

Set the account to the server account.

```
echo 0 | ./set_geth_account.sh
```

## Step 4: [Server side] Start the sealer node

Find your ip address and write to `./config/bootnode_ip`.

```
echo 192.168.1.1 > ./config/bootnode_ip
```

Start the geth client as the sealer node, and output the timing log to `./output/foobar.txt`.

```
echo foobar | ./run_geth_and_update_enode_config.sh
```

The script also start the cloud provider. It wait a few seconds and then get the enode address.

Start the HTTP server powered by Python 3. The HTTP server provide information about the sealer node for the cars.

```
./run_http_server.sh
```

## Notice: Following operations are performed at client side. Don’t run these commands at the server which already has geth running.

## Step 5: [Client side] Get information from the HTTP server

Edit this file:

- `./config/httpserver_url`

Download configuration from the server.

```
./download_config_from_http_server.sh
```

## Step 6: [Client side] Start the node

Select which car account to use. The first one, for example.

```
echo 1 | ./set_geth_account.sh 
```

Start the geth client, and output the timing log to `./output/foobar.txt`.

```
echo foobar | ./run_geth_with_bootnode.sh
```

## Step 7: [Both sides] Stop the nodes

The following command stop the ethereum client **safely**. It sends the `Ctrl+C` signal to the process, instead of just killing them.

```
./kill_geth.sh
```