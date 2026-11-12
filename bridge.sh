#!/bin/bash
#
# Space bridge: a server in between that passes clients' connections on to
# the main servers. A client connects to a port of the bridge, and the bridge
# forwards the connection, TCP or UDP, to a main server's address and port
# (NAT, nothing decrypted or changed on the way).
#
# Run once:   bash <(curl -Ls https://raw.githubusercontent.com/disp911/space-ui/main/bridge.sh)
# Then:       space-bridge          the menu
#             space-bridge apply    put the rules in place (the boot service runs it)
#
# The rules live in chains of their own (SPACE_BRIDGE_*), so the bridge leaves
# the server's other rules, ufw's or Docker's, as they are. The list of bridges
# is kept in /etc/space-bridge/bridges.conf and put back at every boot.

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
blue='\033[0;34m'
plain='\033[0m'

script_url="https://raw.githubusercontent.com/disp911/space-ui/main/bridge.sh"
command_path="/usr/local/bin/space-bridge"
conf_dir="/etc/space-bridge"
conf_file="${conf_dir}/bridges.conf"
unit_file="/etc/systemd/system/space-bridge.service"
# Named to sort last: sysctl files apply in name order and the last one wins,
# and hosting images put their own in, such as 99-sysctl.conf (/etc/sysctl.conf).
sysctl_file="/etc/sysctl.d/99-zz-space-bridge.conf"
# systemd-networkd 256 and later sets forwarding itself and may turn it off
# when it reconfigures a link; older versions ignore the key.
networkd_file="/etc/systemd/networkd.conf.d/99-zz-space-bridge.conf"

LOGE() { echo -e "${red}$*${plain}"; }
LOGI() { echo -e "${green}$*${plain}"; }
LOGW() { echo -e "${yellow}$*${plain}"; }

[[ $EUID -ne 0 ]] && LOGE "Run this script as root (sudo -i first)." && exit 1

if ! command -v systemctl > /dev/null 2>&1; then
    LOGE "The bridge needs systemd to put its rules back at boot; this system has none."
    exit 1
fi

# --- Helpers ----------------------------------------------------------------

is_ipv4() {
    [[ "$1" =~ ^([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})\.([0-9]{1,3})$ ]] || return 1
    local part
    for part in "${BASH_REMATCH[@]:1}"; do
        ((part <= 255)) || return 1
    done
}

is_port() {
    [[ "$1" =~ ^[0-9]+$ ]] && (($1 >= 1 && $1 <= 65535))
}

# resolve prints the IPv4 address of an address or a domain, or nothing.
resolve() {
    if is_ipv4 "$1"; then
        echo "$1"
        return
    fi
    getent ahostsv4 "$1" 2> /dev/null | awk 'NR == 1 {print $1}'
}

# ssh_port prints the port sshd listens on, 22 when it is not found.
ssh_port() {
    local port
    port=$(ss -ltnpH 2> /dev/null | grep '"sshd"' | awk '{print $4}' | sed 's/.*://' | sort -un | head -1)
    echo "${port:-22}"
}

# local_listener names what listens on this server's port $1 (TCP or UDP), if anything.
local_listener() {
    ss -ltnupH "sport = :$1" 2> /dev/null | grep -o '(("[^"]*"' | sed 's/(("//; s/"//' | head -1
}

# public_ip prints this server's public IPv4 address, if it can be found.
public_ip() {
    local url ip
    for url in https://api4.ipify.org https://ipv4.icanhazip.com https://4.ident.me; do
        ip=$(curl -4 -s --max-time 3 "$url" 2> /dev/null | tr -d '[:space:]')
        is_ipv4 "$ip" && echo "$ip" && return
    done
}

proto_name() {
    case "$1" in
        tcp) echo "TCP" ;;
        udp) echo "UDP" ;;
        *) echo "TCP+UDP" ;;
    esac
}

# --- Installing -------------------------------------------------------------

# ensure_iptables installs iptables when the system has none.
ensure_iptables() {
    command -v iptables > /dev/null 2>&1 && return 0
    LOGW "iptables is not installed; installing it..."
    if command -v apt-get > /dev/null 2>&1; then
        apt-get update -q && DEBIAN_FRONTEND=noninteractive apt-get install -y -q iptables
    elif command -v dnf > /dev/null 2>&1; then
        dnf install -y -q iptables
    elif command -v yum > /dev/null 2>&1; then
        yum install -y -q iptables
    elif command -v pacman > /dev/null 2>&1; then
        pacman -S --needed --noconfirm iptables
    fi
    command -v iptables > /dev/null 2>&1 || {
        LOGE "iptables could not be installed; install it and run the script again."
        exit 1
    }
}

# install_self puts the script in place as the space-bridge command, with
# the service that applies the rules at boot.
install_self() {
    mkdir -p "$conf_dir"
    touch "$conf_file"
    if [[ "$(readlink -f "$0" 2> /dev/null)" != "$command_path" ]]; then
        local tmp
        tmp=$(mktemp)
        if curl -fsSL "$script_url" -o "$tmp" 2> /dev/null || curl -4fsSL "$script_url" -o "$tmp"; then
            chmod 755 "$tmp"
            mv -f "$tmp" "$command_path"
        else
            rm -f "$tmp"
            LOGE "The script could not be downloaded from GitHub to ${command_path}."
            exit 1
        fi
    fi
    cat > "$unit_file" << EOF
[Unit]
Description=Space bridge: forwards clients' connections to the main servers
Wants=network-online.target
After=network-online.target ufw.service docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
ExecStart=${command_path} apply

[Install]
WantedBy=multi-user.target
EOF
    systemctl daemon-reload
    systemctl enable space-bridge > /dev/null 2>&1
}

# enable_forwarding turns on IPv4 forwarding, and BBR when asked to.
enable_forwarding() {
    local bbr="$1"
    {
        echo "net.ipv4.ip_forward=1"
        if [[ "$bbr" == "yes" ]]; then
            echo "net.core.default_qdisc=fq"
            echo "net.ipv4.tcp_congestion_control=bbr"
        fi
    } > "$sysctl_file"
    sysctl -q -p "$sysctl_file" > /dev/null
    if [[ -d /etc/systemd/network || -d /run/systemd/netif ]]; then
        mkdir -p "${networkd_file%/*}"
        printf '%s\n' '[Network]' 'IPv4Forwarding=yes' > "$networkd_file"
    fi
}

# --- Rules ------------------------------------------------------------------

# chain TABLE NAME PARENT makes the chain, empty, and jumps to it first thing from PARENT.
chain() {
    iptables -t "$1" -N "$2" 2> /dev/null
    iptables -t "$1" -F "$2"
    iptables -t "$1" -C "$3" -j "$2" 2> /dev/null || iptables -t "$1" -I "$3" 1 -j "$2"
}

drop_chain() {
    while iptables -t "$1" -D "$3" -j "$2" 2> /dev/null; do :; done
    iptables -t "$1" -F "$2" 2> /dev/null
    iptables -t "$1" -X "$2" 2> /dev/null
}

# apply puts the rules of every bridge in the list in place, the old ones gone.
apply() {
    ensure_iptables
    # Again at every boot, after the network is up: the bridge's values win
    # over whatever the hosting set meanwhile.
    if [[ -f "$sysctl_file" ]]; then
        sysctl -q -p "$sysctl_file" > /dev/null
    else
        sysctl -q -w net.ipv4.ip_forward=1 > /dev/null
    fi
    chain nat SPACE_BRIDGE_PRE PREROUTING
    chain nat SPACE_BRIDGE_POST POSTROUTING
    chain filter SPACE_BRIDGE_FWD FORWARD
    chain mangle SPACE_BRIDGE_MSS FORWARD
    # Forward what a bridge rule sent on and the replies to it, nothing else.
    iptables -A SPACE_BRIDGE_FWD -m conntrack --ctstate DNAT -j ACCEPT
    iptables -A SPACE_BRIDGE_FWD -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
    iptables -t mangle -A SPACE_BRIDGE_MSS -p tcp --tcp-flags SYN,RST SYN -j TCPMSS --clamp-mss-to-pmtu

    local proto listen target port ip p failed=0
    while read -r proto listen target port; do
        [[ -z "$proto" || "$proto" == \#* ]] && continue
        ip=$(resolve "$target")
        if [[ -z "$ip" ]]; then
            LOGE "${target}: no IPv4 address found; port ${listen} is not forwarded."
            failed=1
            continue
        fi
        for p in tcp udp; do
            [[ "$proto" != "both" && "$proto" != "$p" ]] && continue
            # Only connections to this server itself, not traffic passing through.
            iptables -t nat -A SPACE_BRIDGE_PRE -p "$p" --dport "$listen" -m addrtype --dst-type LOCAL -j DNAT --to-destination "${ip}:${port}"
            iptables -t nat -A SPACE_BRIDGE_POST -p "$p" -d "$ip" --dport "$port" -j MASQUERADE
        done
    done < "$conf_file"
    return $failed
}

# --- Menu -------------------------------------------------------------------

count_bridges() {
    grep -cvE '^\s*(#|$)' "$conf_file" 2> /dev/null || true
}

show_bridges() {
    local n=0 proto listen target port ip bridge_ip
    if [[ ! -s "$conf_file" ]] || [[ "$(count_bridges)" == "0" ]]; then
        LOGW "No bridges yet."
        return 1
    fi
    bridge_ip=$(public_ip)
    echo -e "${blue}$(printf "  %-2s %-6s %-7s →  %s" "#" "Port" "Proto" "Main server")${plain}"
    while read -r proto listen target port; do
        [[ -z "$proto" || "$proto" == \#* ]] && continue
        n=$((n + 1))
        ip=$(resolve "$target")
        [[ -n "$ip" && "$ip" != "$target" ]] && ip=" (${ip})" || ip=""
        printf "  %-2s %-6s %-7s →  %s:%s%s\n" "$n" "$listen" "$(proto_name "$proto")" "$target" "$port" "$ip"
    done < "$conf_file"
    echo
    echo -e "Clients connect to ${green}${bridge_ip:-<bridge address>}${plain} at the bridge port."
}

# ask_bridge asks for one bridge and adds it to the list.
ask_bridge() {
    local target ip port listen choice proto owner sp
    while true; do
        read -rp "  Main server's address (IP or domain): " target
        target=$(echo "$target" | tr -d '[:space:]')
        ip=$(resolve "$target")
        [[ -n "$ip" ]] && break
        LOGE "  \"${target}\" is not an IPv4 address or a domain with one."
    done
    [[ "$ip" != "$target" ]] && echo "  ${target} is ${ip}"
    while true; do
        read -rp "  Main server's port: " port
        is_port "$port" && break
        LOGE "  A port is a number from 1 to 65535."
    done
    sp=$(ssh_port)
    while true; do
        read -rp "  Port of this bridge clients connect to [default ${port}]: " listen
        listen=${listen:-$port}
        if ! is_port "$listen"; then
            LOGE "  A port is a number from 1 to 65535."
            continue
        fi
        if [[ "$listen" == "$sp" ]]; then
            LOGE "  ${listen} is the SSH port of this server; forwarding it would lock you out."
            continue
        fi
        if awk -v l="$listen" '$2 == l' "$conf_file" | grep -q .; then
            LOGE "  Port ${listen} is already a bridge."
            continue
        fi
        owner=$(local_listener "$listen")
        if [[ -n "$owner" ]]; then
            LOGE "  Port ${listen} is taken by ${owner} on this server."
            continue
        fi
        break
    done
    echo "  Protocol: 1) TCP and UDP  2) TCP only  3) UDP only"
    read -rp "  Choose [default 1]: " choice
    case "$choice" in
        2) proto=tcp ;;
        3) proto=udp ;;
        *) proto=both ;;
    esac
    echo "${proto} ${listen} ${target} ${port}" >> "$conf_file"
    LOGI "  Added: port ${listen} ($(proto_name "$proto")) → ${target}:${port}"
}

add_bridges() {
    local n i bbr=no answer
    while true; do
        read -rp "How many main servers to bridge to? [default 1]: " n
        n=${n:-1}
        [[ "$n" =~ ^[0-9]+$ ]] && ((n >= 1 && n <= 50)) && break
        LOGE "A number from 1 to 50."
    done
    for ((i = 1; i <= n; i++)); do
        echo -e "\n${blue}Server ${i} of ${n}${plain}"
        ask_bridge
    done
    echo
    read -rp "Turn on BBR (TCP congestion control, usually faster)? [Y/n]: " answer
    [[ ! "$answer" =~ ^[Nn] ]] && bbr=yes
    enable_forwarding "$bbr"
    if apply; then
        LOGI "\nThe bridges are in place and come back after a reboot."
    else
        LOGW "\nThe bridges are in place, except the ones above."
    fi
    echo
    show_bridges
    echo -e "\nIn the panel of each main server, add this server's address and port to the inbound's External Proxy, so the subscription gives clients the bridge."
}

remove_bridge() {
    local n
    show_bridges || return
    echo
    read -rp "Number of the bridge to remove (0 to go back): " n
    [[ "$n" =~ ^[0-9]+$ ]] || return
    ((n >= 1 && n <= $(count_bridges))) || return
    awk -v n="$n" '/^[[:space:]]*(#|$)/ {print; next} {k++} k != n' "$conf_file" > "${conf_file}.tmp" && mv -f "${conf_file}.tmp" "$conf_file"
    apply
    LOGI "Bridge ${n} removed."
}

uninstall() {
    local answer
    read -rp "Remove every bridge, the space-bridge command and its boot service? [y/N]: " answer
    [[ "$answer" =~ ^[Yy] ]] || return
    systemctl disable --now space-bridge > /dev/null 2>&1
    rm -f "$unit_file"
    systemctl daemon-reload
    drop_chain nat SPACE_BRIDGE_PRE PREROUTING
    drop_chain nat SPACE_BRIDGE_POST POSTROUTING
    drop_chain filter SPACE_BRIDGE_FWD FORWARD
    drop_chain mangle SPACE_BRIDGE_MSS FORWARD
    rm -rf "$conf_dir" "$sysctl_file" "$networkd_file"
    rm -f "$command_path"
    LOGI "The bridge is removed. IPv4 forwarding stays on until the next reboot."
    exit 0
}

show_rules() {
    iptables -t nat -L SPACE_BRIDGE_PRE -n -v 2> /dev/null || LOGW "No rules in place."
}

menu() {
    while true; do
        echo -e "
${blue}╔──────────────────────────────────────╗${plain}
${blue}│${plain}   ${green}Space bridge${plain}                       ${blue}│${plain}
${blue}│──────────────────────────────────────│${plain}
${blue}│${plain}   ${green}1.${plain} Add bridges                     ${blue}│${plain}
${blue}│${plain}   ${green}2.${plain} Show bridges                    ${blue}│${plain}
${blue}│${plain}   ${green}3.${plain} Remove a bridge                 ${blue}│${plain}
${blue}│${plain}   ${green}4.${plain} Show the rules and traffic      ${blue}│${plain}
${blue}│${plain}   ${green}5.${plain} Uninstall                       ${blue}│${plain}
${blue}│${plain}   ${green}0.${plain} Exit                            ${blue}│${plain}
${blue}╚──────────────────────────────────────╝${plain}"
        echo "Bridges: $(count_bridges)"
        read -rp "Choose: " choice
        case "$choice" in
            1) add_bridges ;;
            2) show_bridges ;;
            3) remove_bridge ;;
            4) show_rules ;;
            5) uninstall ;;
            0) exit 0 ;;
            *) LOGE "Choose a number from the menu." ;;
        esac
    done
}

# --- Start ------------------------------------------------------------------

case "$1" in
    apply)
        [[ -f "$conf_file" ]] || exit 0
        apply
        exit $?
        ;;
    "")
        ensure_iptables
        install_self
        if [[ "$(count_bridges)" == "0" ]]; then
            LOGI "Space bridge: clients connect here and are passed on to your main servers."
            echo
            add_bridges
            echo -e "\nRun ${green}space-bridge${plain} any time to add, show or remove bridges."
            exit 0
        fi
        menu
        ;;
    *)
        echo "Usage: space-bridge        the menu"
        echo "       space-bridge apply  put the rules in place"
        ;;
esac
