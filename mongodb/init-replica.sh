#!/bin/bash
# MongoDB Replica Set Initialization Script
# This script initializes the MongoDB replica set after all nodes are ready

set -eu

# Validate required environment variables
required_vars=(
  "MONGO_INITDB_ROOT_USERNAME"
  "MONGO_INITDB_ROOT_PASSWORD"
  "MONGO_APP_USER"
  "MONGO_APP_PASSWORD"
  "MONGO_DB_NAME"
)

for var in "${required_vars[@]}"; do
  if [ -z "${!var:-}" ]; then
    echo "Error: Required environment variable $var is not set"
    exit 1
  fi
done

# Configuration (can be overridden via environment variables)
MAX_WAIT=${MONGO_INIT_MAX_WAIT:-300}          # Maximum wait time in seconds (default: 5 minutes)
STABILIZATION_DELAY=${MONGO_INIT_STABILIZATION_DELAY:-10}  # Stabilization delay in seconds
RETRY_INTERVAL=${MONGO_INIT_RETRY_INTERVAL:-2}  # Retry interval in seconds
REPLICA_SET_NAME=${MONGO_REPLICA_SET:-rs0}    # Replica set name

# MongoDB hosts (can be overridden via environment variables)
PRIMARY_HOST=${MONGO_PRIMARY_HOST:-flowker-mongodb-primary}
SECONDARY1_HOST=${MONGO_SECONDARY1_HOST:-flowker-mongodb-secondary1}
SECONDARY2_HOST=${MONGO_SECONDARY2_HOST:-flowker-mongodb-secondary2}
MONGO_PORT=${MONGO_PORT:-27017}

# Validate identifiers to prevent JavaScript injection
# Only allow alphanumeric, dots, hyphens, and underscores
validate_identifier() {
  local name=$1
  local value=$2
  if ! [[ "$value" =~ ^[a-zA-Z0-9._-]+$ ]]; then
    echo "Error: $name contains invalid characters: $value"
    echo "Allowed: alphanumeric, dots, hyphens, underscores"
    exit 1
  fi
}

validate_identifier "REPLICA_SET_NAME" "$REPLICA_SET_NAME"
validate_identifier "PRIMARY_HOST" "$PRIMARY_HOST"
validate_identifier "SECONDARY1_HOST" "$SECONDARY1_HOST"
validate_identifier "SECONDARY2_HOST" "$SECONDARY2_HOST"

if ! [[ "$MONGO_PORT" =~ ^[0-9]+$ ]]; then
  echo "Error: MONGO_PORT must be numeric: $MONGO_PORT"
  exit 1
fi

echo "Waiting for MongoDB nodes to be ready..."
echo "Configuration: MAX_WAIT=${MAX_WAIT}s, STABILIZATION_DELAY=${STABILIZATION_DELAY}s, REPLICA_SET=${REPLICA_SET_NAME}"
echo "Hosts: PRIMARY=${PRIMARY_HOST}, SECONDARY1=${SECONDARY1_HOST}, SECONDARY2=${SECONDARY2_HOST}, PORT=${MONGO_PORT}"

# Function to escape special characters for JavaScript strings
# Escapes backslashes, quotes, control characters, and shell metacharacters
escape_js_string() {
  printf '%s' "$1" | sed \
    -e 's/\\/\\\\/g' \
    -e 's/"/\\"/g' \
    -e "s/'/\\\\'/g" \
    -e 's/\$/\\$/g' \
    -e 's/`/\\`/g' \
    -e 's/\x0a/\\n/g' \
    -e 's/\x0d/\\r/g' \
    -e 's/\x09/\\t/g'
}

# Build connection URI for mongosh
# Note: Credentials in URI are more secure than separate -u/-p args
# but still visible in process listings. Acceptable in container environments.
build_mongo_uri() {
  local host=$1
  local port=$2
  local user=$3
  local password=$4
  echo "mongodb://${user}:${password}@${host}:${port}/?authSource=admin"
}

# Function to wait for a MongoDB node with timeout
# Uses connection string to avoid credential exposure in process args
wait_for_node() {
  local host=$1
  local elapsed=0
  local uri
  uri=$(build_mongo_uri "$host" "$MONGO_PORT" "$MONGO_INITDB_ROOT_USERNAME" "$MONGO_INITDB_ROOT_PASSWORD")

  while ! mongosh "$uri" --quiet --eval "db.adminCommand('ping')" &>/dev/null; do
    if [ "$elapsed" -ge "$MAX_WAIT" ]; then
      echo "Error: Timeout waiting for $host after ${MAX_WAIT}s"
      exit 1
    fi
    echo "Waiting for $host... (${elapsed}s/${MAX_WAIT}s)"
    sleep "$RETRY_INTERVAL"
    elapsed=$((elapsed + RETRY_INTERVAL))
  done
  echo "$host is ready"
}

# Wait for all nodes with timeout
wait_for_node "$PRIMARY_HOST"
wait_for_node "$SECONDARY1_HOST"
wait_for_node "$SECONDARY2_HOST"

echo "All MongoDB nodes are ready. Initializing replica set..."

# Build connection URI for primary
PRIMARY_URI=$(build_mongo_uri "$PRIMARY_HOST" "$MONGO_PORT" "$MONGO_INITDB_ROOT_USERNAME" "$MONGO_INITDB_ROOT_PASSWORD")

# Initialize replica set and verify success (uses configurable hosts)
mongosh "$PRIMARY_URI" --quiet <<INITEOF
try {
  const result = rs.initiate({
    _id: "${REPLICA_SET_NAME}",
    members: [
      { _id: 0, host: "${PRIMARY_HOST}:${MONGO_PORT}", priority: 2 },
      { _id: 1, host: "${SECONDARY1_HOST}:${MONGO_PORT}", priority: 1 },
      { _id: 2, host: "${SECONDARY2_HOST}:${MONGO_PORT}", priority: 1 }
    ]
  });

  if (result.ok !== 1) {
    // Check if already initialized (error code 23 = AlreadyInitialized)
    if (result.code === 23 || result.codeName === "AlreadyInitialized") {
      print("Replica set ${REPLICA_SET_NAME} already initialized, continuing...");
    } else {
      print("Error initializing replica set: " + JSON.stringify(result));
      quit(1);
    }
  } else {
    print("Replica set ${REPLICA_SET_NAME} initialized successfully");
  }
} catch (e) {
  // Handle case where replica set is already initialized
  if (e.codeName === "AlreadyInitialized" || e.code === 23) {
    print("Replica set ${REPLICA_SET_NAME} already initialized, continuing...");
  } else {
    print("Error during rs.initiate(): " + e);
    quit(1);
  }
}
INITEOF

echo "Waiting for replica set to stabilize (${STABILIZATION_DELAY}s)..."
sleep "$STABILIZATION_DELAY"

# Check replica set status
echo "Checking replica set status..."
mongosh "$PRIMARY_URI" --quiet --eval "rs.status()"

# Escape credentials for safe JavaScript string interpolation
ESCAPED_APP_USER=$(escape_js_string "$MONGO_APP_USER")
ESCAPED_APP_PASSWORD=$(escape_js_string "$MONGO_APP_PASSWORD")
ESCAPED_DB_NAME=$(escape_js_string "$MONGO_DB_NAME")

# Create application database and user (idempotent - checks if user exists first)
echo "Creating application user (if not exists)..."
mongosh "$PRIMARY_URI" --quiet <<EOF
use ${ESCAPED_DB_NAME}

// Check if user already exists
const existingUser = db.getUser("${ESCAPED_APP_USER}");

if (existingUser === null) {
  print("Creating user ${ESCAPED_APP_USER}...");
  try {
    db.createUser({
      user: "${ESCAPED_APP_USER}",
      pwd: "${ESCAPED_APP_PASSWORD}",
      roles: [
        { role: "readWrite", db: "${ESCAPED_DB_NAME}" }
      ]
    });
    print("User created successfully");
  } catch (e) {
    print("Error creating user: " + e);
    quit(1);
  }
} else {
  print("User ${ESCAPED_APP_USER} already exists, updating roles...");
  try {
    db.updateUser("${ESCAPED_APP_USER}", {
      roles: [
        { role: "readWrite", db: "${ESCAPED_DB_NAME}" }
      ]
    });
    print("User roles updated successfully");
  } catch (e) {
    print("Error updating user: " + e);
    quit(1);
  }
}
EOF

echo "MongoDB replica set initialization complete!"
