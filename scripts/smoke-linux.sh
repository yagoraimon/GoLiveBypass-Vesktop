#!/usr/bin/env bash
set -euo pipefail
root=$(cd -- "$(dirname -- "$0")/.." && pwd)
work=$(mktemp -d)
profile="$work/profile"
mkdir -p "$profile" "$work/vendor"
export XDG_DATA_HOME="$work/data"
export XDG_CONFIG_HOME="$work/config"
trap 'printf quit > "$profile/.golive-vesktop-install-request" 2>/dev/null || true' EXIT
curl -fsSL --proto '=https' 'https://github.com/Vencord/Vesktop/releases/download/v1.6.8/Vesktop-1.6.8.AppImage' -o "$work/vendor/vesktop.AppImage"
echo "1757426d937a6987ae4306e74fac2fe6d3c4ebd4a8895ce3ad6c6ee7ce610d84  $work/vendor/vesktop.AppImage" | sha256sum --check
chmod 700 "$work/vendor/vesktop.AppImage"
(cd "$work/vendor" && ./vesktop.AppImage --appimage-extract > /dev/null)
# Sandbox/GPU flags apply only to the disposable hosted CI desktop.
printf '#!/usr/bin/env bash\nexec %q --no-sandbox --disable-gpu "$@"\n' "$work/vendor/squashfs-root/AppRun" > "$work/vesktop-ci"
chmod 700 "$work/vesktop-ci"
printf '{"firstLaunch":false}\n' > "$profile/state.json"
log="$XDG_DATA_HOME/GoLiveBypass/plugin-vpn/plugin-vpn.log"
ready() {
    local target=$1 count=0
    for _ in $(seq 1 90); do
        if [[ -f "$log" ]]; then count=$(grep -c 'abrindo plugin VPN' "$log" || true); fi
        if (( count >= target )); then
            if grep -q 'falha ao inicializar o controlador VPN' "$log"; then cat "$log"; return 1; fi
            return 0
        fi
        sleep 1
    done
    cat "$log" 2>/dev/null || true
    echo 'Vesktop did not load GoLiveBypass native runtime' >&2
    return 1
}
"$root/release/golive-vesktop-linux-x64" --exe "$work/vesktop-ci" --data-dir "$profile"
ready 1
# The same install command must handle a running fork through app.quit().
"$root/release/golive-vesktop-linux-x64" --exe "$work/vesktop-ci" --data-dir "$profile"
ready 2
printf quit > "$profile/.golive-vesktop-install-request"
alive() {
    local pid
    for pid in $(pgrep -u "$(id -u)" -x vesktop || true); do
        if [[ -e "/proc/$pid/exe" ]]; then return 0; fi
    done
    return 1
}
for _ in $(seq 1 30); do
    if ! alive; then echo 'PASS: real Vesktop startup, packaged native plugin, reinstall and graceful quit.'; exit 0; fi
    sleep 1
done
echo 'Vesktop did not quit through the plugin cleanup path' >&2
exit 1
