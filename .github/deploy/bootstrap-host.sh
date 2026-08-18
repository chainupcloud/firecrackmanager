#!/usr/bin/env bash
set -Eeuo pipefail

require_env() {
  local name="$1"
  if [[ -z "${!name:-}" ]]; then
    echo "Missing required environment variable: ${name}" >&2
    exit 1
  fi
}

require_env APP_BINARY
require_env APP_DATA_DIR
require_env APP_PORT
require_env DEPLOY_HOST
require_env DEPLOY_USER
require_env DEPLOY_SSH_PORT
require_env SSH_KEY_PATH

remote="${DEPLOY_USER}@${DEPLOY_HOST}"
remote_firecracker_version="$(printf '%q' "${FIRECRACKER_VERSION:-}")"
ssh_opts=(
  -i "${SSH_KEY_PATH}"
  -p "${DEPLOY_SSH_PORT}"
  -o BatchMode=yes
  -o StrictHostKeyChecking=yes
)

ssh "${ssh_opts[@]}" "${remote}" \
  "FIRECRACKER_VERSION=${remote_firecracker_version} bash -s" <<'REMOTE_SCRIPT'
set -Eeuo pipefail

need_cmd() {
  local name="$1"
  if ! command -v "${name}" >/dev/null 2>&1; then
    echo "Missing required command after bootstrap: ${name}" >&2
    exit 1
  fi
}

install_packages() {
  local packages=("$@")

  if command -v apt-get >/dev/null 2>&1; then
    DEBIAN_FRONTEND=noninteractive apt-get update -qq
    DEBIAN_FRONTEND=noninteractive apt-get install -y --no-install-recommends "${packages[@]}"
    return
  fi

  if command -v dnf >/dev/null 2>&1; then
    dnf install -y "${packages[@]}"
    return
  fi

  if command -v yum >/dev/null 2>&1; then
    yum install -y "${packages[@]}"
    return
  fi

  echo "Unsupported host package manager; install these packages first: ${packages[*]}" >&2
  exit 1
}

install_firecracker() {
  local arch api_url download_url tmp_dir firecracker_bin jailer_bin

  arch="$(uname -m)"
  if [[ "${arch}" != "x86_64" ]]; then
    echo "Unsupported host architecture: ${arch}; current workflow builds linux/amd64 only" >&2
    exit 1
  fi

  if [[ -x /usr/sbin/firecracker && -x /usr/sbin/jailer ]]; then
    local installed_version

    installed_version="$(/usr/sbin/firecracker --version)"
    echo "${installed_version}"
    /usr/sbin/jailer --version >/dev/null 2>&1 || true

    if [[ -z "${FIRECRACKER_VERSION}" || "${installed_version}" == *"${FIRECRACKER_VERSION}"* ]]; then
      echo "Firecracker and jailer already installed"
      return
    fi

    echo "Installed Firecracker does not match requested ${FIRECRACKER_VERSION}; upgrading"
  fi

  if [[ -n "${FIRECRACKER_VERSION}" ]]; then
    api_url="https://api.github.com/repos/firecracker-microvm/firecracker/releases/tags/${FIRECRACKER_VERSION}"
  else
    api_url="https://api.github.com/repos/firecracker-microvm/firecracker/releases/latest"
  fi

  download_url="$(
    curl --fail --silent --show-error --location "${api_url}" |
      jq -r --arg arch "${arch}" '
        first(.assets[] | select(.name | test($arch + ".*\\.tgz$")) | .browser_download_url) // empty
      '
  )"

  if [[ -z "${download_url}" || "${download_url}" == "null" ]]; then
    echo "Cannot find Firecracker release asset for ${arch} from ${api_url}" >&2
    exit 1
  fi

  tmp_dir="$(mktemp -d)"
  trap 'rm -rf "${tmp_dir}"' RETURN

  curl --fail --silent --show-error --location "${download_url}" --output "${tmp_dir}/firecracker.tgz"
  mkdir -p "${tmp_dir}/extract"
  tar -xzf "${tmp_dir}/firecracker.tgz" -C "${tmp_dir}/extract"

  firecracker_bin="$(find "${tmp_dir}/extract" -type f -name "firecracker-v*-${arch}" -print -quit)"
  jailer_bin="$(find "${tmp_dir}/extract" -type f -name "jailer-v*-${arch}" -print -quit)"

  if [[ -z "${firecracker_bin}" || -z "${jailer_bin}" ]]; then
    echo "Firecracker archive does not contain expected firecracker/jailer binaries" >&2
    exit 1
  fi

  install -m 755 "${firecracker_bin}" /usr/sbin/firecracker
  install -m 755 "${jailer_bin}" /usr/sbin/jailer
  /usr/sbin/firecracker --version
}

if [[ "$(id -u)" -ne 0 ]]; then
  echo "Bootstrap must run as root on the hardware host" >&2
  exit 1
fi

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemd/systemctl is required on the hardware host" >&2
  exit 1
fi

if [[ ! -d /run/systemd/system ]]; then
  echo "systemd is not running as the host init system" >&2
  exit 1
fi

if [[ ! -c /dev/kvm ]]; then
  echo "/dev/kvm is required for Firecracker microVMs" >&2
  exit 1
fi

# 新宿主机只补齐明确依赖；不在这里做应用发布，发布复用 deploy.sh。
install_packages ca-certificates curl gzip iptables jq tar
need_cmd curl
need_cmd iptables
need_cmd jq
need_cmd tar

install_firecracker

systemctl daemon-reload
/usr/sbin/firecracker --version
test -x /usr/sbin/jailer
REMOTE_SCRIPT

SSH_KEY_PATH="${SSH_KEY_PATH}" .github/deploy/deploy.sh

ssh "${ssh_opts[@]}" "${remote}" \
  "APP_PORT='${APP_PORT}' bash -s" <<'REMOTE_SCRIPT'
set -Eeuo pipefail

systemctl is-enabled --quiet firecrackmanager
systemctl is-active --quiet firecrackmanager
test -x /usr/local/bin/firecrackmanager
test -x /usr/sbin/firecracker
test -x /usr/sbin/jailer
/usr/sbin/firecracker --version
curl --fail --silent --show-error --max-time 10 "http://0.0.0.0:${APP_PORT}/" >/dev/null
REMOTE_SCRIPT
