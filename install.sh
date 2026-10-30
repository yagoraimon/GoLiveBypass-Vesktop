#!/usr/bin/env bash
set -euo pipefail
if [[ "$(uname -s)" != Linux || "$(uname -m)" != x86_64 ]]; then
    echo 'Esta release atende Linux x64. macOS fica para uma versao futura.' >&2
    exit 1
fi
if [[ "${EUID}" == 0 ]]; then
    echo 'Execute como seu usuario normal, sem sudo.' >&2
    exit 1
fi
base='https://github.com/yagoraimon/GoLiveBypass-Vesktop/releases/latest/download'
asset='golive-vesktop-linux-x64'
temporary=$(mktemp -d)
trap 'rm -rf -- "$temporary"' EXIT
echo 'Baixando o plugin pronto para Vesktop...'
curl --fail --location --silent --show-error --proto '=https' "$base/$asset" -o "$temporary/$asset"
curl --fail --location --silent --show-error --proto '=https' "$base/$asset.sha256" -o "$temporary/$asset.sha256"
(cd "$temporary" && sha256sum --check "$asset.sha256")
chmod 700 "$temporary/$asset"
"$temporary/$asset"
