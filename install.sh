#!/usr/bin/env bash
set -euo pipefail

PACKAGE_NAME="${BOIDS_PACKAGE:-boids-sdk}"
VERSION="${BOIDS_VERSION:-}"
METHOD="${BOIDS_INSTALL_METHOD:-auto}"

if [ -n "$VERSION" ]; then
  NPM_SPEC="${PACKAGE_NAME}@${VERSION}"
  PYPI_SPEC="${PACKAGE_NAME}==${VERSION}"
else
  NPM_SPEC="${PACKAGE_NAME}"
  PYPI_SPEC="${PACKAGE_NAME}"
fi

log() {
  printf '%s\n' "$*"
}

warn() {
  printf 'warning: %s\n' "$*" >&2
}

has() {
  command -v "$1" >/dev/null 2>&1
}

install_with_npm() {
  log "Installing Boids CLI with npm: ${NPM_SPEC}"
  npm install -g "$NPM_SPEC"
}

install_with_pipx() {
  log "Installing Boids CLI with pipx: ${PYPI_SPEC}"
  pipx install --force "$PYPI_SPEC"
}

install_with_pip_user() {
  local python_bin="$1"
  log "Installing Boids CLI with pip --user: ${PYPI_SPEC}"
  "$python_bin" -m pip install --user --upgrade "$PYPI_SPEC"
}

verify_install() {
  if has boids; then
    log "Boids CLI installed:"
    boids --help | sed -n '1,8p'
    return 0
  fi

  warn "Boids CLI was installed, but 'boids' is not on PATH yet."
  warn "Restart your shell or add your npm/pip user bin directory to PATH."
}

case "$METHOD" in
  npm)
    install_with_npm
    verify_install
    ;;
  pipx)
    install_with_pipx
    verify_install
    ;;
  pip)
    if has python3; then
      install_with_pip_user python3
    elif has python; then
      install_with_pip_user python
    else
      warn "Python was not found."
      exit 1
    fi
    verify_install
    ;;
  auto)
    if has npm && install_with_npm; then
      verify_install
      exit 0
    fi

    if has pipx && install_with_pipx; then
      verify_install
      exit 0
    fi

    if has python3 && install_with_pip_user python3; then
      verify_install
      exit 0
    fi

    if has python && install_with_pip_user python; then
      verify_install
      exit 0
    fi

    warn "Could not install boids-sdk. Install Node.js/npm, pipx, or Python first."
    exit 1
    ;;
  *)
    warn "Unknown BOIDS_INSTALL_METHOD: ${METHOD}"
    warn "Use one of: auto, npm, pipx, pip"
    exit 1
    ;;
esac
