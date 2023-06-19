#!/bin/bash

ACCOUNTS_DIR="accounts"
PERUN_CONTRACTS_DIR="contracts"
SYSTEM_SCRIPTS_DIR="system_scripts"
DEVNET_DIR="$PWD"

genesis=$(cat $ACCOUNTS_DIR/genesis-2.txt | awk '/testnet/ { count++; if (count == 2) print $2}')

cd $PERUN_CONTRACTS_DIR

if [ -d "migrations/dev" ]; then
  rm -rf "migrations/dev"
fi

expect << EOF
spawn capsule deploy --address $genesis --api "http://127.0.0.1:8114" --fee 0.01
expect "Confirm deployment? (Yes/No)"
send "Yes\r"
expect "Password:"
send "\r"
expect eof
EOF

# Fetch default contracts:
cd $DEVNET_DIR

if [ -d "$SYSTEM_SCRIPTS_DIR" ]; then
  rm -rf "$SYSTEM_SCRIPTS_DIR"
fi

mkdir -p "$SYSTEM_SCRIPTS_DIR"
## jq will interpret the code_hash and tx_hash as numbers, so we need to wrap them in quotes.
## The index must also be a string value, but yaml does not support hex values as a top level block argument
## so we have to do that in a second pass...
ckb-cli util genesis-scripts \
  | sed 's/code_hash: \(.*\)/code_hash: \"\1\"/; s/tx_hash: \(.*\)/tx_hash: \"\1\"/' \
  | yq . \
  | sed 's/"index": \(.*\),/echo "\\"index\\": $(python -c "print(\\\"\\\\\\"{}\\\\\\"\\\".format(hex(\1)))"),";/e' \
  | jq . > "$SYSTEM_SCRIPTS_DIR/default_scripts.json"
