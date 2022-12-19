#!/bin/bash
CURRENT_DIR=$1
rm -rf ${CURRENT_DIR}/genproto
for x in $(find ${CURRENT_DIR}/* -type d); do
  protoc -I=${x} -I=${CURRENT_DIR}/ -I /usr/local/include --go_out=${CURRENT_DIR} \
   --go-grpc_out=${CURRENT_DIR} ${x}/*.proto
rm -rf ${CURRENT_DIR}/genproto
done