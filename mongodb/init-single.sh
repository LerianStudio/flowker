#!/bin/bash
# MongoDB single-node replica set initialization for local development (no auth required)
set -eu

REPLICA_SET=${MONGO_REPLICA_SET:-rs0}
# Hostname recorded inside rs.initiate (drivers will use this)
# Default to flowker-mongodb for container environments; override with MONGO_PRIMARY_HOST for localhost dev
MONGO_HOST=${MONGO_PRIMARY_HOST:-flowker-mongodb}
# Hostname used by this init container to reach mongod
MONGO_CONNECT_HOST=${MONGO_CONNECT_HOST:-flowker-mongodb}
MONGO_PORT=${MONGO_PORT:-27017}

APP_USER=${MONGO_APP_USER:-}
APP_PASS=${MONGO_APP_PASSWORD:-}
APP_DB=${MONGO_DB_NAME:-}

uri="mongodb://${MONGO_CONNECT_HOST}:${MONGO_PORT}/"

echo "Waiting for Mongo to be ready at ${MONGO_HOST}:${MONGO_PORT}..."
ready=0
for _ in {1..60}; do
  if mongosh "$uri" --quiet --eval "db.adminCommand('ping')" >/dev/null 2>&1; then
    ready=1
    break
  fi
  sleep 2
done

if [ "$ready" -ne 1 ]; then
  echo "Mongo was not ready after 120s; aborting init." >&2
  exit 1
fi

echo "Initiating single-node replica set ${REPLICA_SET}..."
mongosh "$uri" --quiet <<EOF
try {
  const status = rs.status();
  if (status.ok === 1) {
    print("Replica set already initiated.");
  }
} catch (e) {
  const result = rs.initiate({
    _id: "${REPLICA_SET}",
    members: [{ _id: 0, host: "${MONGO_HOST}:${MONGO_PORT}" }]
  });
  if (result.ok !== 1 && result.code !== 23) {
    throw result;
  }
}
EOF

if [ -n "$APP_USER" ] && [ -n "$APP_PASS" ] && [ -n "$APP_DB" ]; then
  echo "Creating application user..."
  mongosh "$uri" --quiet <<EOF
use ${APP_DB}
if (db.getUser("${APP_USER}") == null) {
  db.createUser({
    user: "${APP_USER}",
    pwd: "${APP_PASS}",
    roles: [{ role: "readWrite", db: "${APP_DB}" }]
  });
  print("User created");
} else {
  print("User already exists");
}
EOF
else
  echo "Skipping app user creation (APP_USER/APP_PASS/APP_DB not set)"
fi

echo "Mongo single-node setup completed."
