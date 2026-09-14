#!/usr/bin/with-contenv bashio

bashio::log.info "Starting go-trmnl..."

# ── Data directory ────────────────────────────────────────────────────────────
# /config is the add-on's persistent config directory (maps to addon_config).
DATA_DIR="/config/go-trmnl"
mkdir -p "${DATA_DIR}"
export TRMNL_DATA_DIR="${DATA_DIR}"

# ── Base URL ──────────────────────────────────────────────────────────────────
BASE_URL=$(bashio::config 'base_url')
if bashio::var.is_empty "${BASE_URL}"; then
    # Fall back to the HA host's IP on the mapped port
    HOST_IP=$(bashio::network.ipv4_address)
    BASE_URL="http://${HOST_IP}:8080"
    bashio::log.warning "base_url not set — using auto-detected: ${BASE_URL}"
fi
export TRMNL_BASE_URL="${BASE_URL}"

# ── Admin password ────────────────────────────────────────────────────────────
ADMIN_PASSWORD=$(bashio::config 'admin_password')
if ! bashio::var.is_empty "${ADMIN_PASSWORD}"; then
    export TRMNL_ADMIN_PASSWORD="${ADMIN_PASSWORD}"
fi

# ── Encryption ────────────────────────────────────────────────────────────────
SECRET_KEY=$(bashio::config 'secret_key')
if ! bashio::var.is_empty "${SECRET_KEY}"; then
    export TRMNL_SECRET_KEY="${SECRET_KEY}"
fi

NO_ENCRYPTION=$(bashio::config 'no_encryption')
if bashio::var.true "${NO_ENCRYPTION}"; then
    export TRMNL_NO_ENCRYPTION="true"
fi

# ── Device auth ───────────────────────────────────────────────────────────────
DISABLE_AUTH=$(bashio::config 'disable_device_auth')
if bashio::var.true "${DISABLE_AUTH}"; then
    export TRMNL_DISABLE_DEVICE_AUTH="true"
fi

bashio::log.info "Base URL  : ${TRMNL_BASE_URL}"
bashio::log.info "Data dir  : ${TRMNL_DATA_DIR}"
bashio::log.info "Admin UI  : ${TRMNL_BASE_URL}/admin"

exec /usr/local/bin/trmnld
