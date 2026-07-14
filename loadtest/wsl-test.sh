#!/bin/bash
HOST_IP=$(grep nameserver /etc/resolv.conf | awk '{print $2}')
echo "Host IP: $HOST_IP"
echo "Testing connectivity..."
curl -s -o /dev/null -w "HTTP %{http_code}\n" "http://${HOST_IP}:8080/ingest" \
  -X POST -H "Content-Type: application/json" \
  -d '[{"service":"test","level":"info","message":"ping"}]'
