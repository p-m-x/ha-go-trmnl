#!/usr/bin/with-contenv bashio

bashio::log.info "Starting TRMNL Display add-on..."

# Read user config via bashio
ENTITIES=$(bashio::config 'entities | join(",")')
REFRESH_RATE=$(bashio::config 'refresh_rate')
TITLE=$(bashio::config 'title')
COLUMNS=$(bashio::config 'columns')

# Inside HA add-ons the supervisor proxies HA REST API.
# SUPERVISOR_TOKEN is injected automatically by the supervisor.
export HA_URL="http://supervisor/core"
export HA_TOKEN="${SUPERVISOR_TOKEN}"
export HA_ENTITIES="${ENTITIES}"
export REFRESH_RATE="${REFRESH_RATE}"
export DISPLAY_TITLE="${TITLE}"
export DISPLAY_COLUMNS="${COLUMNS}"

# BASE_URL must be reachable by the TRMNL device.
# If HA ingress is not used, the device connects directly on the mapped port.
HOST_IP=$(bashio::addon.ip_address)
export BASE_URL="http://${HOST_IP}:8080"

bashio::log.info "Entities  : ${ENTITIES}"
bashio::log.info "Refresh   : ${REFRESH_RATE}s"
bashio::log.info "Base URL  : ${BASE_URL}"

exec /ha-trmnld
