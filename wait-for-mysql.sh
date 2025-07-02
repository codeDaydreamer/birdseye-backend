#!/bin/sh
# wait-for-mysql.sh

set -e

host="$1"
port="$2"

echo "Waiting for MySQL at $host:$port..."

while ! nc -z "$host" "$port"; do
  echo "MySQL is unavailable - sleeping"
  sleep 2
done

echo "MySQL is up - executing command"
exec "${@:3}"
