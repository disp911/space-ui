#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

#Add some basic function here
function LOGD() {
    echo -e "${yellow}[DEG] $* ${plain}"
}

function LOGE() {
    echo -e "${red}[ERR] $* ${plain}"
}

function LOGI() {
    echo -e "${green}[INF] $* ${plain}"
}

# Port helpers: detect listener and owning process (best effort)
is_port_in_use() {
    local port="$1"
    if command -v ss > /dev/null 2>&1; then
        ss -ltn 2> /dev/null | awk -v p=":${port}$" '$4 ~ p {exit 0} END {exit 1}'
        return
    fi
    if command -v netstat > /dev/null 2>&1; then
        netstat -lnt 2> /dev/null | awk -v p=":${port} " '$4 ~ p {exit 0} END {exit 1}'
        return
    fi
    if command -v lsof > /dev/null 2>&1; then
        lsof -nP -iTCP:${port} -sTCP:LISTEN > /dev/null 2>&1 && return 0
    fi
    return 1
}

# Simple helpers for domain/IP validation
is_ipv4() {
    [[ "$1" =~ ^([0-9]{1,3}\.){3}[0-9]{1,3}$ ]] && return 0 || return 1
}
is_ipv6() {
    [[ "$1" =~ : ]] && return 0 || return 1
}
is_ip() {
    is_ipv4 "$1" || is_ipv6 "$1"
}
is_domain() {
    [[ "$1" =~ ^([A-Za-z0-9](-*[A-Za-z0-9])*\.)+(xn--[a-z0-9]{2,}|[A-Za-z]{2,})$ ]] && return 0 || return 1
}

# check root
[[ $EUID -ne 0 ]] && LOGE "ERROR: You must be root to run this script! \n" && exit 1

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    echo "Failed to check the system OS, please contact the author!" >&2
    exit 1
fi
[[ "$1" == site-* || "$1" == help ]] || echo "The OS release is: $release"

os_version=""
# The major version only: 24 for Ubuntu 24.04, 12 for Debian 12.
os_version=$(grep "^VERSION_ID" /etc/os-release | cut -d '=' -f2 | tr -d '"' | cut -d '.' -f1)

# Declare Variables
space_ui_folder="${SPACE_UI_MAIN_FOLDER:=/usr/local/space-ui}"
space_ui_service="${SPACE_UI_SERVICE:=/etc/systemd/system}"
log_folder="${SPACE_UI_LOG_FOLDER:=/var/log/space-ui}"
mkdir -p "${log_folder}"
iplimit_log_path="${log_folder}/space-ui-ipl.log"
iplimit_banned_log_path="${log_folder}/space-ui-ipl-banned.log"

confirm() {
    if [[ $# > 1 ]]; then
        echo && read -rp "$1 [Default $2]: " temp
        if [[ "${temp}" == "" ]]; then
            temp=$2
        fi
    else
        read -rp "$1 [y/n]: " temp
    fi
    if [[ "${temp}" == "y" || "${temp}" == "Y" ]]; then
        return 0
    else
        return 1
    fi
}

confirm_restart() {
    confirm "Restart the panel, Attention: Restarting the panel will also restart xray" "y"
    if [[ $? == 0 ]]; then
        restart
    else
        show_menu
    fi
}

before_show_menu() {
    echo && echo -n -e "${yellow}Press enter to return to the main menu: ${plain}" && read -r temp
    show_menu
}

install() {
    bash <(curl -Ls https://raw.githubusercontent.com/disp911/space-ui/main/install.sh)
    if [[ $? == 0 ]]; then
        if [[ $# == 0 ]]; then
            start
        else
            start 0
        fi
    fi
}

update() {
    confirm "This function will update all space-ui components to the latest version, and the data will not be lost. Do you want to continue?" "y"
    if [[ $? != 0 ]]; then
        LOGE "Cancelled"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 0
    fi
    bash <(curl -Ls https://raw.githubusercontent.com/disp911/space-ui/main/update.sh)
    if [[ $? == 0 ]]; then
        LOGI "Update is complete, Panel has automatically restarted "
        before_show_menu
    fi
}

# fetch downloads a URL to a file, over IPv4 when the default way fails.
fetch() {
    curl -fsLRo "$1" "$2" 2> /dev/null || curl -4fsLRo "$1" "$2"
}

# pkg_install installs packages with the system's package manager, without
# upgrading the rest of the system.
pkg_install() {
    case "${release}" in
        ubuntu | debian | armbian)
            apt-get update && apt-get install -y "$@"
            ;;
        fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
            dnf -y install "$@"
            ;;
        centos)
            if [[ "${VERSION_ID}" =~ ^7 ]]; then
                yum -y install "$@"
            else
                dnf -y install "$@"
            fi
            ;;
        arch | manjaro | parch)
            pacman -S --needed --noconfirm "$@"
            ;;
        alpine)
            apk add "$@"
            ;;
        *)
            return 1
            ;;
    esac
}

# update_menu puts back the menu of the installed version, so the menu and
# the panel always come from one release.
update_menu() {
    local version tmp
    version=$(${space_ui_folder}/space-ui -v 2> /dev/null)
    if [[ -z "$version" ]]; then
        LOGE "The installed version is unknown."
        [[ $# == 0 ]] && before_show_menu
        return 1
    fi
    confirm "This replaces the menu with the one of the installed version v${version}." "y"
    if [[ $? != 0 ]]; then
        LOGE "Cancelled"
        [[ $# == 0 ]] && before_show_menu
        return 0
    fi
    tmp=$(mktemp)
    if ! fetch "$tmp" "https://raw.githubusercontent.com/disp911/space-ui/v${version}/space-ui.sh"; then
        rm -f "$tmp"
        LOGE "Failed to download the menu, please be sure that your server can access GitHub."
        [[ $# == 0 ]] && before_show_menu
        return 1
    fi
    chmod +x "$tmp"
    cp -f "$tmp" ${space_ui_folder}/space-ui.sh
    mv -f "$tmp" /usr/bin/space-ui
    LOGI "The menu of v${version} is in place. Run space-ui again."
    exit 0
}

legacy_version() {
    echo -n "Enter the panel version (like 3.0.0): "
    read -r tag_version
    tag_version=${tag_version#v}

    if ! [[ "$tag_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
        echo "A version is three numbers, like 3.0.0. Exiting."
        exit 1
    fi
    if [[ "$(printf '%s\n' 3.0.0 "$tag_version" | sort -V | head -n1)" != "3.0.0" ]]; then
        echo "Versions before 3.0.0 are not Space UI. Exiting."
        exit 1
    fi

    echo "Downloading and installing panel version $tag_version..."
    bash <(curl -Ls "https://raw.githubusercontent.com/disp911/space-ui/v${tag_version}/install.sh") "v${tag_version}"
}

# Function to handle the deletion of the script file
delete_script() {
    rm "$0" # Remove the script file itself
    exit 1
}

uninstall() {
    confirm "Are you sure you want to uninstall the panel? xray will also uninstalled!" "n"
    if [[ $? != 0 ]]; then
        if [[ $# == 0 ]]; then
            show_menu
        fi
        return 0
    fi

    if [[ $release == "alpine" ]]; then
        rc-service space-ui stop
        rc-update del space-ui
        rm /etc/init.d/space-ui -f
    else
        systemctl stop space-ui
        systemctl disable space-ui
        rm ${space_ui_service}/space-ui.service -f
        systemctl daemon-reload
        systemctl reset-failed
    fi

    rm /etc/space-ui/ -rf
    rm ${space_ui_folder}/ -rf

    echo ""
    echo -e "Uninstalled Successfully.\n"
    echo "If you need to install this panel again, you can use below command:"
    echo -e "${green}bash <(curl -Ls https://raw.githubusercontent.com/disp911/space-ui/main/install.sh)${plain}"
    echo ""
    # Trap the SIGTERM signal
    trap delete_script SIGTERM
    delete_script
}

reset_user() {
    confirm "Are you sure to reset the username and password of the panel?" "n"
    if [[ $? != 0 ]]; then
        if [[ $# == 0 ]]; then
            show_menu
        fi
        return 0
    fi

    read -rp "Please set the login username [default is a random username]: " config_account
    [[ -z $config_account ]] && config_account=$(gen_random_string 10)
    read -rp "Please set the login password [default is a random password]: " config_password
    [[ -z $config_password ]] && config_password=$(gen_random_string 18)

    read -rp "Do you want to disable currently configured two-factor authentication? (y/n): " twoFactorConfirm
    if [[ $twoFactorConfirm != "y" && $twoFactorConfirm != "Y" ]]; then
        ${space_ui_folder}/space-ui setting -username "${config_account}" -password "${config_password}" -resetTwoFactor false > /dev/null 2>&1
    else
        ${space_ui_folder}/space-ui setting -username "${config_account}" -password "${config_password}" -resetTwoFactor true > /dev/null 2>&1
        echo -e "Two factor authentication has been disabled."
    fi

    echo -e "Panel login username has been reset to: ${green} ${config_account} ${plain}"
    echo -e "Panel login password has been reset to: ${green} ${config_password} ${plain}"
    echo -e "${green} Please use the new login username and password to access the panel. Also remember them! ${plain}"
    confirm_restart
}

gen_random_string() {
    local length="$1"
    openssl rand -base64 $((length * 2)) \
        | tr -dc 'a-zA-Z0-9' \
        | head -c "$length"
}

reset_webbasepath() {
    echo -e "${yellow}Resetting Web Base Path${plain}"

    read -rp "Are you sure you want to reset the web base path? (y/n): " confirm
    if [[ $confirm != "y" && $confirm != "Y" ]]; then
        echo -e "${yellow}Operation canceled.${plain}"
        return
    fi

    config_webBasePath=$(gen_random_string 18)

    # Apply the new web base path setting
    ${space_ui_folder}/space-ui setting -webBasePath "${config_webBasePath}" > /dev/null 2>&1

    echo -e "Web base path has been reset to: ${green}${config_webBasePath}${plain}"
    echo -e "${green}Please use the new web base path to access the panel.${plain}"
    restart
}

check_config() {
    local info=$(${space_ui_folder}/space-ui setting -show true)
    if [[ $? != 0 ]]; then
        LOGE "get current settings error, please check logs"
        show_menu
        return
    fi
    LOGI "${info}"

    site_status
}

start() {
    check_status
    if [[ $? == 0 ]]; then
        echo ""
        LOGI "Panel is running, No need to start again, If you need to restart, please select restart"
    else
        if [[ $release == "alpine" ]]; then
            rc-service space-ui start
        else
            systemctl start space-ui
        fi
        sleep 2
        check_status
        if [[ $? == 0 ]]; then
            LOGI "space-ui Started Successfully"
        else
            LOGE "panel Failed to start, Probably because it takes longer than two seconds to start, Please check the log information later"
        fi
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

stop() {
    check_status
    if [[ $? == 1 ]]; then
        echo ""
        LOGI "Panel stopped, No need to stop again!"
    else
        if [[ $release == "alpine" ]]; then
            rc-service space-ui stop
        else
            systemctl stop space-ui
        fi
        sleep 2
        check_status
        if [[ $? == 1 ]]; then
            LOGI "space-ui and xray stopped successfully"
        else
            LOGE "Panel stop failed, Probably because the stop time exceeds two seconds, Please check the log information later"
        fi
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

restart() {
    if [[ $release == "alpine" ]]; then
        rc-service space-ui restart
    else
        systemctl restart space-ui
    fi
    sleep 2
    check_status
    if [[ $? == 0 ]]; then
        LOGI "space-ui and xray Restarted successfully"
    else
        LOGE "Panel restart failed, Probably because it takes longer than two seconds to start, Please check the log information later"
    fi
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

restart_xray() {
    systemctl reload space-ui
    LOGI "xray-core Restart signal sent successfully, Please check the log information to confirm whether xray restarted successfully"
    sleep 2
    show_xray_status
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

status() {
    if [[ $release == "alpine" ]]; then
        rc-service space-ui status
    else
        systemctl status space-ui -l
    fi
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

enable() {
    if [[ $release == "alpine" ]]; then
        rc-update add space-ui default
    else
        systemctl enable space-ui
    fi
    if [[ $? == 0 ]]; then
        LOGI "space-ui Set to boot automatically on startup successfully"
    else
        LOGE "space-ui Failed to set Autostart"
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

disable() {
    if [[ $release == "alpine" ]]; then
        rc-update del space-ui
    else
        systemctl disable space-ui
    fi
    if [[ $? == 0 ]]; then
        LOGI "space-ui Autostart Cancelled successfully"
    else
        LOGE "space-ui Failed to cancel autostart"
    fi

    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

show_log() {
    if [[ $release == "alpine" ]]; then
        grep -F 'space-ui[' /var/log/messages
    else
        journalctl -u space-ui -e --no-pager -f -p debug
    fi
    if [[ $# == 0 ]]; then
        before_show_menu
    fi
}

bbr_menu() {
    echo -e "${green}\t1.${plain} Enable BBR"
    echo -e "${green}\t2.${plain} Disable BBR"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice
    case "$choice" in
        0)
            show_menu
            ;;
        1)
            enable_bbr
            bbr_menu
            ;;
        2)
            disable_bbr
            bbr_menu
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            bbr_menu
            ;;
    esac
}

disable_bbr() {

    if [[ $(sysctl -n net.ipv4.tcp_congestion_control) != "bbr" ]] || [[ ! $(sysctl -n net.core.default_qdisc) =~ ^(fq|cake)$ ]]; then
        echo -e "${yellow}BBR is not currently enabled.${plain}"
        before_show_menu
    fi

    if [ -f "/etc/sysctl.d/99-bbr-space-ui.conf" ]; then
        old_settings=$(head -1 /etc/sysctl.d/99-bbr-space-ui.conf | tr -d '#')
        sysctl -w net.core.default_qdisc="${old_settings%:*}"
        sysctl -w net.ipv4.tcp_congestion_control="${old_settings#*:}"
        rm /etc/sysctl.d/99-bbr-space-ui.conf
        sysctl --system
    else
        # Replace BBR with CUBIC configurations
        if [ -f "/etc/sysctl.conf" ]; then
            sed -i 's/net.core.default_qdisc=fq/net.core.default_qdisc=pfifo_fast/' /etc/sysctl.conf
            sed -i 's/net.ipv4.tcp_congestion_control=bbr/net.ipv4.tcp_congestion_control=cubic/' /etc/sysctl.conf
            sysctl -p
        fi
    fi

    if [[ $(sysctl -n net.ipv4.tcp_congestion_control) != "bbr" ]]; then
        echo -e "${green}BBR has been replaced with CUBIC successfully.${plain}"
    else
        echo -e "${red}Failed to replace BBR with CUBIC. Please check your system configuration.${plain}"
    fi
}

enable_bbr() {
    if [[ $(sysctl -n net.ipv4.tcp_congestion_control) == "bbr" ]] && [[ $(sysctl -n net.core.default_qdisc) =~ ^(fq|cake)$ ]]; then
        echo -e "${green}BBR is already enabled!${plain}"
        before_show_menu
    fi

    # Enable BBR
    if [ -d "/etc/sysctl.d/" ]; then
        {
            echo "#$(sysctl -n net.core.default_qdisc):$(sysctl -n net.ipv4.tcp_congestion_control)"
            echo "net.core.default_qdisc = fq"
            echo "net.ipv4.tcp_congestion_control = bbr"
        } > "/etc/sysctl.d/99-bbr-space-ui.conf"
        if [ -f "/etc/sysctl.conf" ]; then
            # Backup old settings from sysctl.conf, if any
            sed -i 's/^net.core.default_qdisc/# &/' /etc/sysctl.conf
            sed -i 's/^net.ipv4.tcp_congestion_control/# &/' /etc/sysctl.conf
        fi
        sysctl --system
    else
        sed -i '/net.core.default_qdisc/d' /etc/sysctl.conf
        sed -i '/net.ipv4.tcp_congestion_control/d' /etc/sysctl.conf
        echo "net.core.default_qdisc=fq" | tee -a /etc/sysctl.conf
        echo "net.ipv4.tcp_congestion_control=bbr" | tee -a /etc/sysctl.conf
        sysctl -p
    fi

    # Verify that BBR is enabled
    if [[ $(sysctl -n net.ipv4.tcp_congestion_control) == "bbr" ]]; then
        echo -e "${green}BBR has been enabled successfully.${plain}"
    else
        echo -e "${red}Failed to enable BBR. Please check your system configuration.${plain}"
    fi
}

# 0: running, 1: not running, 2: not installed
check_status() {
    if [[ $release == "alpine" ]]; then
        if [[ ! -f /etc/init.d/space-ui ]]; then
            return 2
        fi
        if [[ $(rc-service space-ui status | grep -F 'status: started' -c) == 1 ]]; then
            return 0
        else
            return 1
        fi
    else
        if [[ ! -f ${space_ui_service}/space-ui.service ]]; then
            return 2
        fi
        temp=$(systemctl status space-ui | grep Active | awk '{print $3}' | cut -d "(" -f2 | cut -d ")" -f1)
        if [[ "${temp}" == "running" ]]; then
            return 0
        else
            return 1
        fi
    fi
}

check_enabled() {
    if [[ $release == "alpine" ]]; then
        if [[ $(rc-update show | grep -F 'space-ui' | grep default -c) == 1 ]]; then
            return 0
        else
            return 1
        fi
    else
        temp=$(systemctl is-enabled space-ui)
        if [[ "${temp}" == "enabled" ]]; then
            return 0
        else
            return 1
        fi
    fi
}

check_uninstall() {
    check_status
    if [[ $? != 2 ]]; then
        echo ""
        LOGE "Panel installed, Please do not reinstall"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 1
    else
        return 0
    fi
}

check_install() {
    check_status
    if [[ $? == 2 ]]; then
        echo ""
        LOGE "Please install the panel first"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 1
    else
        return 0
    fi
}

show_status() {
    check_status
    case $? in
        0)
            echo -e "Panel state: ${green}Running${plain}"
            show_enable_status
            ;;
        1)
            echo -e "Panel state: ${yellow}Not Running${plain}"
            show_enable_status
            ;;
        2)
            echo -e "Panel state: ${red}Not Installed${plain}"
            ;;
    esac
    show_xray_status
}

show_enable_status() {
    check_enabled
    if [[ $? == 0 ]]; then
        echo -e "Start automatically: ${green}Yes${plain}"
    else
        echo -e "Start automatically: ${red}No${plain}"
    fi
}

check_xray_status() {
    count=$(ps -ef | grep "xray-linux" | grep -v "grep" | wc -l)
    if [[ count -ne 0 ]]; then
        return 0
    else
        return 1
    fi
}

show_xray_status() {
    check_xray_status
    if [[ $? == 0 ]]; then
        echo -e "xray state: ${green}Running${plain}"
    else
        echo -e "xray state: ${red}Not Running${plain}"
    fi
}

# ufw_allows says whether ufw has a rule letting TCP port $1 in; $2 names
# the same port as ufw's services do, such as OpenSSH|ssh.
ufw_allows() {
    ufw show added 2> /dev/null | grep -Eq "^ufw allow (in )?(${1}(/tcp)?|${2:-$1})$"
}

# firewall_enable turns ufw on only once SSH and the panel stay reachable:
# it offers to open their ports first.
firewall_enable() {
    local missing=() names=() port
    port=$(ssh_port)
    ufw_allows "$port" "OpenSSH|ssh" || { missing+=("$port"); names+=("SSH ${port}"); }
    if [[ -x ${space_ui_folder}/space-ui ]]; then
        if [[ "$(site_value "website mode")" == "reality" ]]; then
            port=443
        else
            port=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^port: .+' | awk '{print $2}')
        fi
        if [[ -n "$port" ]] && ! ufw_allows "$port" "https"; then
            missing+=("$port")
            names+=("panel ${port}")
        fi
    fi
    if [[ ${#missing[@]} -gt 0 ]]; then
        echo -e "${yellow}The firewall would block: ${names[*]}.${plain}"
        [[ "${names[0]}" == SSH* ]] && echo -e "${yellow}Without SSH you lose access to the server.${plain}"
        if confirm "Open these ports first?" "y"; then
            for port in "${missing[@]}"; do
                ufw allow "${port}/tcp"
            done
        elif ! confirm "Enable the firewall anyway?" "n"; then
            echo -e "${yellow}Cancelled.${plain}"
            return
        fi
    fi
    echo -e "${yellow}Inbounds on other ports stay blocked until you open them (Open Ports).${plain}"
    ufw enable
}

firewall_menu() {
    if ! command -v ufw > /dev/null 2>&1; then
        echo -e "${yellow}ufw is not installed; the menu manages ufw only.${plain}"
        before_show_menu
        return
    fi
    echo -e "${green}\t1.${plain} Port List [numbered]"
    echo -e "${green}\t2.${plain} ${green}Open${plain} Ports"
    echo -e "${green}\t3.${plain} ${red}Delete${plain} Ports from List"
    echo -e "${green}\t4.${plain} ${green}Enable${plain} Firewall"
    echo -e "${green}\t5.${plain} ${red}Disable${plain} Firewall"
    echo -e "${green}\t6.${plain} Firewall Status"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice
    case "$choice" in
        0)
            show_menu
            ;;
        1)
            ufw status numbered
            firewall_menu
            ;;
        2)
            open_ports
            firewall_menu
            ;;
        3)
            delete_ports
            firewall_menu
            ;;
        4)
            firewall_enable
            firewall_menu
            ;;
        5)
            ufw disable
            firewall_menu
            ;;
        6)
            ufw status verbose
            firewall_menu
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            firewall_menu
            ;;
    esac
}

open_ports() {
    # Prompt the user to enter the ports they want to open
    read -rp "Enter the ports you want to open (e.g. 80,443,2053 or range 400-500): " ports

    # Check if the input is valid
    if ! [[ $ports =~ ^([0-9]+|[0-9]+-[0-9]+)(,([0-9]+|[0-9]+-[0-9]+))*$ ]]; then
        echo "Error: Invalid input. Please enter a comma-separated list of ports or a range of ports (e.g. 80,443,2053 or 400-500)." >&2
        exit 1
    fi

    # Open the specified ports using ufw
    IFS=',' read -ra PORT_LIST <<< "$ports"
    for port in "${PORT_LIST[@]}"; do
        if [[ $port == *-* ]]; then
            # Split the range into start and end ports
            start_port=$(echo $port | cut -d'-' -f1)
            end_port=$(echo $port | cut -d'-' -f2)
            # Open the port range
            ufw allow $start_port:$end_port/tcp
            ufw allow $start_port:$end_port/udp
        else
            # Open the single port
            ufw allow "$port"
        fi
    done

    # Confirm that the ports are opened
    echo "Opened the specified ports:"
    for port in "${PORT_LIST[@]}"; do
        if [[ $port == *-* ]]; then
            start_port=$(echo $port | cut -d'-' -f1)
            end_port=$(echo $port | cut -d'-' -f2)
            # Check if the port range has been successfully opened
            (ufw status | grep -q "$start_port:$end_port") && echo "$start_port-$end_port"
        else
            # Check if the individual port has been successfully opened
            (ufw status | grep -q "$port") && echo "$port"
        fi
    done
}

delete_ports() {
    # Display current rules with numbers
    echo "Current UFW rules:"
    ufw status numbered

    # Ask the user how they want to delete rules
    echo "Do you want to delete rules by:"
    echo "1) Rule numbers"
    echo "2) Ports"
    read -rp "Enter your choice (1 or 2): " choice

    if [[ $choice -eq 1 ]]; then
        # Deleting by rule numbers
        read -rp "Enter the rule numbers you want to delete (1, 2, etc.): " rule_numbers

        # Validate the input
        if ! [[ $rule_numbers =~ ^([0-9]+)(,[0-9]+)*$ ]]; then
            echo "Error: Invalid input. Please enter a comma-separated list of rule numbers." >&2
            exit 1
        fi

        # Split numbers into an array
        IFS=',' read -ra RULE_NUMBERS <<< "$rule_numbers"
        for rule_number in "${RULE_NUMBERS[@]}"; do
            # Delete the rule by number
            ufw delete "$rule_number" || echo "Failed to delete rule number $rule_number"
        done

        echo "Selected rules have been deleted."

    elif [[ $choice -eq 2 ]]; then
        # Deleting by ports
        read -rp "Enter the ports you want to delete (e.g. 80,443,2053 or range 400-500): " ports

        # Validate the input
        if ! [[ $ports =~ ^([0-9]+|[0-9]+-[0-9]+)(,([0-9]+|[0-9]+-[0-9]+))*$ ]]; then
            echo "Error: Invalid input. Please enter a comma-separated list of ports or a range of ports (e.g. 80,443,2053 or 400-500)." >&2
            exit 1
        fi

        # Split ports into an array
        IFS=',' read -ra PORT_LIST <<< "$ports"
        for port in "${PORT_LIST[@]}"; do
            if [[ $port == *-* ]]; then
                # Split the port range
                start_port=$(echo $port | cut -d'-' -f1)
                end_port=$(echo $port | cut -d'-' -f2)
                # Delete the port range
                ufw delete allow $start_port:$end_port/tcp
                ufw delete allow $start_port:$end_port/udp
            else
                # Delete a single port
                ufw delete allow "$port"
            fi
        done

        # Confirmation of deletion
        echo "Deleted the specified ports:"
        for port in "${PORT_LIST[@]}"; do
            if [[ $port == *-* ]]; then
                start_port=$(echo $port | cut -d'-' -f1)
                end_port=$(echo $port | cut -d'-' -f2)
                # Check if the port range has been deleted
                (ufw status | grep -q "$start_port:$end_port") || echo "$start_port-$end_port"
            else
                # Check if the individual port has been deleted
                (ufw status | grep -q "$port") || echo "$port"
            fi
        done
    else
        echo "${red}Error:${plain} Invalid choice. Please enter 1 or 2." >&2
        exit 1
    fi
}

update_all_geofiles() {
    update_geofiles "main"
    update_geofiles "RU"
}

update_geofiles() {
    case "${1}" in
        "main")
            dat_files=(geoip geosite)
            dat_source="Loyalsoldier/v2ray-rules-dat"
            ;;
        "RU")
            dat_files=(geoip_RU geosite_RU)
            dat_source="runetfreedom/russia-v2ray-rules-dat"
            ;;
    esac
    for dat in "${dat_files[@]}"; do
        # Remove suffix for remote filename (e.g., geoip_RU -> geoip)
        remote_file="${dat%%_*}"
        curl -fLRo ${space_ui_folder}/bin/${dat}.dat -z ${space_ui_folder}/bin/${dat}.dat \
            https://github.com/${dat_source}/releases/latest/download/${remote_file}.dat
    done
}

update_geo() {
    echo -e "${green}\t1.${plain} Loyalsoldier (geoip.dat, geosite.dat)"
    echo -e "${green}\t2.${plain} runetfreedom (geoip_RU.dat, geosite_RU.dat)"
    echo -e "${green}\t3.${plain} All"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice

    case "$choice" in
        0)
            show_menu
            ;;
        1)
            update_geofiles "main"
            echo -e "${green}Loyalsoldier datasets have been updated successfully!${plain}"
            restart
            ;;
        2)
            update_geofiles "RU"
            echo -e "${green}runetfreedom datasets have been updated successfully!${plain}"
            restart
            ;;
        3)
            update_all_geofiles
            echo -e "${green}All geo files have been updated successfully!${plain}"
            restart
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            update_geo
            ;;
    esac

    before_show_menu
}

# --- Website, panel address and certificate ---------------------------------
# The panel runs one of two ways: behind VLESS REALITY on 443 (selfsteal),
# where its website, the panel at the secret path and the subscriptions share
# 443 with the VPN; or on a port of its own with the same website at the
# root, leaving 443 free. The panel obtains its certificate itself.

# site_value prints a value `space-ui site` shows, such as "website mode".
site_value() {
    ${space_ui_folder}/space-ui site 2> /dev/null | sed -n "s/^$1: //p"
}

# server_ips prints this server's public IPv4 and IPv6 addresses, those found.
server_ips() {
    local ip
    for url in https://api4.ipify.org https://ipv4.icanhazip.com https://4.ident.me; do
        ip=$(curl -4 -s --max-time 3 "$url" 2> /dev/null | tr -d '[:space:]')
        is_ipv4 "$ip" && echo "$ip" && break
    done
    for url in https://api6.ipify.org https://ipv6.icanhazip.com https://6.ident.me; do
        ip=$(curl -6 -s --max-time 3 "$url" 2> /dev/null | tr -d '[:space:]')
        is_ipv6 "$ip" && echo "$ip" && break
    done
}

# port_owner names what listens on a TCP port other than the panel and its
# Xray, as "name pid", or prints nothing.
port_owner() {
    command -v ss > /dev/null 2>&1 || return 0
    ss -ltnpH "sport = :$1" 2> /dev/null \
        | grep -o '(("[^"]*",pid=[0-9]*' | sed 's/(("//; s/",pid=/ /' \
        | grep -Ev '^(space-ui|xray-linux-[a-z0-9]*) ' | head -1
}

# service_of prints the systemd unit a process runs in, if any.
service_of() {
    command -v systemctl > /dev/null 2>&1 || return 0
    systemctl status "$1" 2> /dev/null | head -1 | grep -o '[^ ]*\.service' | head -1
}

# free_port_443 makes sure nothing else holds TCP 443 for REALITY: it offers
# to stop what does. It returns 0 when 443 is free, and 2 when it stays taken
# and the panel goes on a port of its own instead.
free_port_443() {
    local owner name pid unit choice
    owner=$(port_owner 443)
    [[ -z "$owner" ]] && return 0
    name=${owner% *}
    pid=${owner#* }
    unit=$(service_of "$pid")
    echo -e "${yellow}TCP port 443 is taken by ${name} (pid ${pid}${unit:+, ${unit}}).${plain}"
    echo -e "${green}1.${plain} Free port 443: stop ${unit:-${name}} and keep it from starting again"
    echo -e "${green}2.${plain} Leave ${name} on 443 and put the website and the panel on a port of your choice"
    read -rp "Choose an option [default 2]: " choice
    [[ "$choice" == "1" ]] || return 2
    if [[ -n "$unit" ]]; then
        systemctl disable --now "$unit" > /dev/null 2>&1
    else
        kill "$pid" 2> /dev/null
    fi
    sleep 2
    if [[ -n "$(port_owner 443)" ]]; then
        LOGE "Port 443 is still taken; the website and the panel go on a port of your choice."
        return 2
    fi
    LOGI "Port 443 is free now."
    return 0
}

# ask_domain asks for a domain and checks it points at this server; with
# "optional" an empty answer is fine. It sets SITE_NAME.
ask_domain() {
    local optional="$1" ips addrs choice
    ips=$(server_ips)
    while true; do
        if [[ -n "$optional" ]]; then
            read -rp "Domain of the panel (leave empty to use the IP address): " SITE_NAME
        else
            read -rp "Domain of the website and the panel (its A/AAAA record points at this server): " SITE_NAME
        fi
        SITE_NAME=$(echo "${SITE_NAME}" | tr -d '[:space:]' | tr 'A-Z' 'a-z')
        [[ -z "$SITE_NAME" && -n "$optional" ]] && return 0
        if ! is_domain "$SITE_NAME"; then
            LOGE "\"${SITE_NAME}\" is not a domain name."
            continue
        fi
        addrs=$(${space_ui_folder}/space-ui site -lookup "$SITE_NAME" 2> /dev/null)
        if [[ -n "$ips" ]] && echo "$addrs" | grep -qxF -f <(echo "$ips"); then
            LOGI "${SITE_NAME} points at this server."
            return 0
        fi
        echo -e "${yellow}${SITE_NAME} does not point at this server yet.${plain}"
        echo -e "  ${SITE_NAME}: $(echo ${addrs:-no address})"
        echo -e "  this server: $(echo ${ips:-not found})"
        echo -e "${yellow}The certificate is checked at the domain's address, so it fails until the record points here.${plain}"
        echo -e "${green}1.${plain} Enter another domain"
        echo -e "${green}2.${plain} Go on with ${SITE_NAME} (the record has just been changed)"
        read -rp "Choose an option [default 1]: " choice
        [[ "$choice" == "2" ]] && return 0
    done
}

# warn_port_80 tells what a busy port 80 means for the certificate.
warn_port_80() {
    local owner
    owner=$(port_owner 80)
    [[ -z "$owner" ]] && return 0
    echo -e "${yellow}Port 80 is taken by ${owner% *}: the certificate is checked over 443 only.${plain}"
    return 1
}

# ask_port asks for the panel's port, the current one by default, and sets
# it. The website and the panel answer on it.
ask_port() {
    local port current
    current=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^port: .+' | awk '{print $2}')
    while true; do
        read -rp "Port of the panel [default ${current}]: " port
        port=${port:-$current}
        if ! [[ "$port" =~ ^[0-9]+$ ]] || ((port < 1 || port > 65535)); then
            LOGE "A port is a number from 1 to 65535."
            continue
        fi
        if [[ -n "$(port_owner "$port")" ]]; then
            LOGE "Port ${port} is taken by $(port_owner "$port" | cut -d' ' -f1)."
            continue
        fi
        break
    done
    ${space_ui_folder}/space-ui setting -port "$port" > /dev/null
    SITE_PORT=$port
}

# setup_selfsteal puts the website and the panel on 443 behind REALITY.
# It returns 2 when 443 stays taken and the panel goes on a port of its own.
setup_selfsteal() {
    free_port_443 || return 2
    ask_domain
    warn_port_80
    local out
    out=$(${space_ui_folder}/space-ui selfsteal -domain "$SITE_NAME" 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "$out"
        LOGE "The website could not be put behind REALITY."
        return 1
    fi
    SITE_CLIENT=$(echo "$out" | grep -E '^(client link|subscription):')
    LOGI "The website, the panel and the subscriptions will answer on 443 behind REALITY."
    return 0
}

# setup_own_port puts the panel and its website on a port of its own.
setup_own_port() {
    local port choice cert key
    ask_port
    port=$SITE_PORT
    ask_domain optional
    if [[ -z "$SITE_NAME" ]]; then
        SITE_NAME=$(server_ips | head -1)
        if [[ -z "$SITE_NAME" ]]; then
            LOGE "The server's public address was not found; enter a domain instead."
            ask_domain
        else
            echo -e "${yellow}The panel gets a certificate for ${SITE_NAME}: Let's Encrypt gives an address one for six days, renewed every few days over port 80 or 443.${plain}"
        fi
    fi
    echo -e "${green}1.${plain} The panel obtains the certificate itself"
    echo -e "${green}2.${plain} Certificate files of my own"
    read -rp "Choose an option [default 1]: " choice
    if [[ "$choice" == "2" ]]; then
        while true; do
            read -rp "Certificate file (fullchain): " cert
            read -rp "Private key file: " key
            [[ -s "$cert" && -s "$key" ]] && break
            LOGE "Both files must exist and not be empty."
        done
    else
        cert=""
        if [[ -n "$(port_owner 80)" && -n "$(port_owner 443)" ]]; then
            echo -e "${yellow}Ports 80 and 443 are both taken: the authorities cannot check this server, and the panel serves a self-signed certificate until one of them is free.${plain}"
        fi
    fi
    if [[ -n "$cert" ]]; then
        ${space_ui_folder}/space-ui cert -webCert "$cert" -webCertKey "$key" > /dev/null
    else
        ${space_ui_folder}/space-ui cert -reset > /dev/null
    fi
    local out
    out=$(${space_ui_folder}/space-ui site -mode port -domain "$SITE_NAME" 2>&1)
    if [[ $? -ne 0 ]]; then
        echo "$out"
        LOGE "The panel could not be put on port ${port}."
        return 1
    fi
    SITE_CLIENT=""
    LOGI "The website, the panel and the subscriptions will answer on port ${port}."
    return 0
}

# site_setup asks how the panel is to be reached and sets it up. It changes
# the settings only; the caller restarts the panel.
site_setup() {
    local choice
    echo -e "${green}How is the panel to be reached?${plain}"
    echo -e "${green}1.${plain} Website and panel on a port of your choice (with a domain or the IP address)"
    echo -e "${green}2.${plain} Website and panel on 443 behind VLESS REALITY (selfsteal; needs a domain)"
    read -rp "Choose an option [default 1]: " choice
    if [[ "$choice" == "2" ]]; then
        setup_selfsteal
        case $? in
            0) return 0 ;;
            1) return 1 ;;
        esac
    fi
    setup_own_port
}

# site_url prints the panel's address from the settings.
site_url() {
    local mode name port path
    mode=$(site_value "website mode")
    name=$(site_value "website name")
    port=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^port: .+' | awk '{print $2}')
    path=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^webBasePath: .+' | awk '{print $2}')
    is_ipv6 "$name" && name="[${name}]"
    if [[ "$mode" == "reality" ]]; then
        echo "https://${name}${path}"
    elif [[ -z "$name" ]]; then
        echo "http(s)://<server address>:${port}${path}"
    elif [[ "$port" == "443" ]]; then
        echo "https://${name}${path}"
    else
        echo "https://${name}:${port}${path}"
    fi
}

# site_status prints where the panel is and whether its certificate is
# trusted yet; with "wait" it gives the certificate a minute and a half.
site_status() {
    local url tries=1 code
    url=$(site_url)
    [[ "$1" == "wait" ]] && tries=18
    if [[ "$url" == https://* ]]; then
        for ((i = 0; i < tries; i++)); do
            code=$(curl -s -o /dev/null -w '%{http_code}' --max-time 5 "$url")
            [[ "$code" != "000" ]] && break
            ((i + 1 < tries)) && sleep 5
        done
        if [[ "$code" != "000" ]]; then
            LOGI "The panel answers with a trusted certificate."
        elif [[ "$(curl -sk -o /dev/null -w '%{http_code}' --max-time 5 "$url")" != "000" ]]; then
            echo -e "${yellow}The panel answers with a temporary self-signed certificate, so the browser warns; it keeps trying to get a real one (see: space-ui log).${plain}"
        else
            LOGE "The panel does not answer at ${url} yet (see: space-ui log)."
        fi
    fi
    echo -e "${green}Access URL: ${url}${plain}"
}

# site_menu is the menu of the website, the panel's address and certificate.
site_menu() {
    local mode
    mode=$(site_value "website mode")
    echo -e "${green}Website and certificate${plain}"
    echo -e "  mode: ${mode}, name: $(site_value "website name")"
    echo -e "${green}\t1.${plain} Show the panel's address"
    echo -e "${green}\t2.${plain} Set up again: a port of its own or REALITY on 443"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice
    case "$choice" in
        1)
            site_status
            site_menu
            ;;
        2)
            if site_setup; then
                restart 0
                site_status wait
                [[ -n "$SITE_CLIENT" ]] && echo "$SITE_CLIENT"
            fi
            site_menu
            ;;
        *) show_menu ;;
    esac
}

ip_validation() {
    ipv6_regex="^(([0-9a-fA-F]{1,4}:){7,7}[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,7}:|([0-9a-fA-F]{1,4}:){1,6}:[0-9a-fA-F]{1,4}|([0-9a-fA-F]{1,4}:){1,5}(:[0-9a-fA-F]{1,4}){1,2}|([0-9a-fA-F]{1,4}:){1,4}(:[0-9a-fA-F]{1,4}){1,3}|([0-9a-fA-F]{1,4}:){1,3}(:[0-9a-fA-F]{1,4}){1,4}|([0-9a-fA-F]{1,4}:){1,2}(:[0-9a-fA-F]{1,4}){1,5}|[0-9a-fA-F]{1,4}:((:[0-9a-fA-F]{1,4}){1,6})|:((:[0-9a-fA-F]{1,4}){1,7}|:)|fe80:(:[0-9a-fA-F]{0,4}){0,4}%[0-9a-zA-Z]{1,}|::(ffff(:0{1,4}){0,1}:){0,1}((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])|([0-9a-fA-F]{1,4}:){1,4}:((25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9])\.){3,3}(25[0-5]|(2[0-4]|1{0,1}[0-9]){0,1}[0-9]))$"
    ipv4_regex="^((25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]?|0)\.){3}(25[0-5]|2[0-4][0-9]|1[0-9][0-9]|[1-9][0-9]?|0)$"
}

iplimit_main() {
    echo -e "\n${green}\t1.${plain} Install Fail2ban and configure IP Limit"
    echo -e "${green}\t2.${plain} Change Ban Duration"
    echo -e "${green}\t3.${plain} Unban Everyone"
    echo -e "${green}\t4.${plain} Ban Logs"
    echo -e "${green}\t5.${plain} Ban an IP Address"
    echo -e "${green}\t6.${plain} Unban an IP Address"
    echo -e "${green}\t7.${plain} Real-Time Logs"
    echo -e "${green}\t8.${plain} Service Status"
    echo -e "${green}\t9.${plain} Service Restart"
    echo -e "${green}\t10.${plain} Remove IP Limit or Fail2ban"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice
    case "$choice" in
        0)
            show_menu
            ;;
        1)
            confirm "Proceed with installation of Fail2ban & IP Limit?" "y"
            if [[ $? == 0 ]]; then
                install_iplimit
            else
                iplimit_main
            fi
            ;;
        2)
            read -rp "Please enter new Ban Duration in Minutes [default 30]: " NUM
            if [[ $NUM =~ ^[0-9]+$ ]]; then
                create_iplimit_jails ${NUM}
                if [[ $release == "alpine" ]]; then
                    rc-service fail2ban restart
                else
                    systemctl restart fail2ban
                fi
            else
                echo -e "${red}${NUM} is not a number! Please, try again.${plain}"
            fi
            iplimit_main
            ;;
        3)
            confirm "Proceed with Unbanning everyone from IP Limit jail?" "y"
            if [[ $? == 0 ]]; then
                fail2ban-client reload --restart --unban space-ui-ipl
                truncate -s 0 "${iplimit_banned_log_path}"
                echo -e "${green}All users Unbanned successfully.${plain}"
                iplimit_main
            else
                echo -e "${yellow}Cancelled.${plain}"
            fi
            iplimit_main
            ;;
        4)
            show_banlog
            iplimit_main
            ;;
        5)
            read -rp "Enter the IP address you want to ban: " ban_ip
            ip_validation
            if [[ $ban_ip =~ $ipv4_regex || $ban_ip =~ $ipv6_regex ]]; then
                fail2ban-client set space-ui-ipl banip "$ban_ip"
                echo -e "${green}IP Address ${ban_ip} has been banned successfully.${plain}"
            else
                echo -e "${red}Invalid IP address format! Please try again.${plain}"
            fi
            iplimit_main
            ;;
        6)
            read -rp "Enter the IP address you want to unban: " unban_ip
            ip_validation
            if [[ $unban_ip =~ $ipv4_regex || $unban_ip =~ $ipv6_regex ]]; then
                fail2ban-client set space-ui-ipl unbanip "$unban_ip"
                echo -e "${green}IP Address ${unban_ip} has been unbanned successfully.${plain}"
            else
                echo -e "${red}Invalid IP address format! Please try again.${plain}"
            fi
            iplimit_main
            ;;
        7)
            tail -f /var/log/fail2ban.log
            iplimit_main
            ;;
        8)
            service fail2ban status
            iplimit_main
            ;;
        9)
            if [[ $release == "alpine" ]]; then
                rc-service fail2ban restart
            else
                systemctl restart fail2ban
            fi
            iplimit_main
            ;;
        10)
            remove_iplimit
            iplimit_main
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            iplimit_main
            ;;
    esac
}

install_iplimit() {
    if ! command -v fail2ban-client &> /dev/null; then
        echo -e "${green}Fail2ban is not installed. Installing now...!${plain}\n"

        # nftables comes along: recent fail2ban packages ban with nftables by
        # default (the sshd jail), and minimal images lack it.
        if [[ "${release}" == "debian" && "${os_version}" -ge 12 ]]; then
            pkg_install fail2ban nftables python3-systemd
        elif [[ "${release}" == "centos" && "${VERSION_ID}" =~ ^7 ]]; then
            pkg_install epel-release && pkg_install fail2ban nftables
        else
            pkg_install fail2ban nftables
        fi
        if [[ $? -ne 0 ]] && ! command -v fail2ban-client &> /dev/null; then
            echo -e "${red}Fail2ban could not be installed on ${release}. Please install it manually.${plain}\n"
            before_show_menu
            return 1
        fi

        if ! command -v fail2ban-client &> /dev/null; then
            echo -e "${red}Fail2ban installation failed.${plain}\n"
            before_show_menu
            return 1
        fi

        echo -e "${green}Fail2ban installed successfully!${plain}\n"
    else
        echo -e "${yellow}Fail2ban is already installed.${plain}\n"
    fi

    # The IP Limit jail bans with iptables; minimal images may have nftables only.
    if ! command -v iptables &> /dev/null; then
        echo -e "${green}Installing iptables for the IP Limit jail...${plain}\n"
        pkg_install iptables
    fi

    echo -e "${green}Configuring IP Limit...${plain}\n"

    # make sure there's no conflict for jail files
    iplimit_remove_conflicts

    # Check if log file exists
    if ! test -f "${iplimit_banned_log_path}"; then
        touch ${iplimit_banned_log_path}
    fi

    # Check if service log file exists so fail2ban won't return error
    if ! test -f "${iplimit_log_path}"; then
        touch ${iplimit_log_path}
    fi

    # Create the iplimit jail files
    # we didn't pass the bantime here to use the default value
    create_iplimit_jails

    # Launching fail2ban
    if [[ $release == "alpine" ]]; then
        if [[ $(rc-service fail2ban status | grep -F 'status: started' -c) == 0 ]]; then
            rc-service fail2ban start
        else
            rc-service fail2ban restart
        fi
        rc-update add fail2ban
    else
        systemctl restart fail2ban
        systemctl enable fail2ban
        sleep 2
        # Fail2ban 1.0 on Python 3.12 (Ubuntu 24.04) needs asyncore and
        # asynchat, which Python no longer has; the distribution packages them.
        if ! fail2ban-client ping > /dev/null 2>&1 && [[ "${release}" =~ ^(ubuntu|debian|armbian)$ ]]; then
            echo -e "${yellow}Fail2ban does not start; installing the Python modules it needs...${plain}\n"
            pkg_install python3-pyasyncore python3-pyasynchat && systemctl restart fail2ban
            sleep 2
        fi
    fi

    if fail2ban-client ping > /dev/null 2>&1; then
        echo -e "${green}IP Limit installed and configured successfully!${plain}\n"
    else
        echo -e "${red}Fail2ban does not run; see: journalctl -u fail2ban${plain}\n"
    fi
    before_show_menu
}

# remove_iplimit takes the IP Limit jail away, and on request Fail2ban itself:
# only its package goes, the rest of /etc/fail2ban and other jails' settings stay.
remove_iplimit() {
    echo -e "${green}\t1.${plain} Remove IP Limit only (Fail2ban stays)"
    echo -e "${green}\t2.${plain} Remove IP Limit and uninstall Fail2ban"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " num
    case "$num" in
        1 | 2)
            local others
            others=$(fail2ban-client status 2> /dev/null | sed -n 's/.*Jail list:[[:space:]]*//p' | tr -d ' ' | tr ',' '\n' | grep -vx 'space-ui-ipl' | paste -sd ' ')
            if [[ "$num" == "2" && -n "$others" ]]; then
                echo -e "${yellow}Fail2ban also guards: ${others}. Uninstalling it ends that protection too.${plain}"
                confirm "Uninstall Fail2ban anyway?" "n" || {
                    remove_iplimit
                    return
                }
            fi
            rm -f /etc/fail2ban/filter.d/space-ui-ipl.conf
            rm -f /etc/fail2ban/action.d/space-ui-ipl.conf
            rm -f /etc/fail2ban/jail.d/space-ui-ipl.conf
            if [[ "$num" == "1" ]]; then
                if [[ $release == "alpine" ]]; then
                    rc-service fail2ban restart
                else
                    systemctl restart fail2ban
                fi
                echo -e "${green}IP Limit removed successfully!${plain}\n"
                before_show_menu
                return
            fi
            if [[ $release == "alpine" ]]; then
                rc-service fail2ban stop
                rc-update del fail2ban
            else
                systemctl disable --now fail2ban
            fi
            case "${release}" in
                ubuntu | debian | armbian)
                    apt-get remove -y fail2ban
                    ;;
                fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
                    dnf remove -y fail2ban
                    ;;
                centos)
                    if [[ "${VERSION_ID}" =~ ^7 ]]; then
                        yum remove -y fail2ban
                    else
                        dnf remove -y fail2ban
                    fi
                    ;;
                arch | manjaro | parch)
                    pacman -R --noconfirm fail2ban
                    ;;
                alpine)
                    apk del fail2ban
                    ;;
                *)
                    echo -e "${red}Unsupported operating system. Please uninstall Fail2ban manually.${plain}\n"
                    before_show_menu
                    return 1
                    ;;
            esac
            echo -e "${green}IP Limit removed and Fail2ban uninstalled; its other settings in /etc/fail2ban are kept.${plain}\n"
            before_show_menu
            ;;
        0)
            show_menu
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            remove_iplimit
            ;;
    esac
}

show_banlog() {
    local system_log="/var/log/fail2ban.log"

    echo -e "${green}Checking ban logs...${plain}\n"

    if [[ $release == "alpine" ]]; then
        if [[ $(rc-service fail2ban status | grep -F 'status: started' -c) == 0 ]]; then
            echo -e "${red}Fail2ban service is not running!${plain}\n"
            return 1
        fi
    else
        if ! systemctl is-active --quiet fail2ban; then
            echo -e "${red}Fail2ban service is not running!${plain}\n"
            return 1
        fi
    fi

    if [[ -f "$system_log" ]]; then
        echo -e "${green}Recent system ban activities from fail2ban.log:${plain}"
        grep "space-ui-ipl" "$system_log" | grep -E "Ban|Unban" | tail -n 10 || echo -e "${yellow}No recent system ban activities found${plain}"
        echo ""
    fi

    if [[ -f "${iplimit_banned_log_path}" ]]; then
        echo -e "${green}IP limit ban log entries:${plain}"
        if [[ -s "${iplimit_banned_log_path}" ]]; then
            grep -v "INIT" "${iplimit_banned_log_path}" | tail -n 10 || echo -e "${yellow}No ban entries found${plain}"
        else
            echo -e "${yellow}Ban log file is empty${plain}"
        fi
    else
        echo -e "${red}Ban log file not found at: ${iplimit_banned_log_path}${plain}"
    fi

    echo -e "\n${green}Current jail status:${plain}"
    fail2ban-client status space-ui-ipl || echo -e "${yellow}Unable to get jail status${plain}"
}

create_iplimit_jails() {
    # Use default bantime if not passed => 30 minutes
    local bantime="${1:-30}"

    # Uncomment 'allowipv6 = auto' in fail2ban.conf
    sed -i 's/#allowipv6 = auto/allowipv6 = auto/g' /etc/fail2ban/fail2ban.conf

    # On Debian 12+ fail2ban's default backend should be changed to systemd
    if [[ "${release}" == "debian" && ${os_version} -ge 12 ]]; then
        sed -i '0,/action =/s/backend = auto/backend = systemd/' /etc/fail2ban/jail.conf
    fi

    cat << EOF > /etc/fail2ban/jail.d/space-ui-ipl.conf
[space-ui-ipl]
enabled=true
backend=auto
filter=space-ui-ipl
action=space-ui-ipl
logpath=${iplimit_log_path}
maxretry=2
findtime=32
bantime=${bantime}m
EOF

    cat << EOF > /etc/fail2ban/filter.d/space-ui-ipl.conf
[Definition]
datepattern = ^%%Y/%%m/%%d %%H:%%M:%%S
failregex   = \[LIMIT_IP\]\s*Email\s*=\s*<F-USER>.+</F-USER>\s*\|\|\s*Disconnecting OLD IP\s*=\s*<ADDR>\s*\|\|\s*Timestamp\s*=\s*\d+
ignoreregex =
EOF

    cat << EOF > /etc/fail2ban/action.d/space-ui-ipl.conf
[INCLUDES]
before = iptables-allports.conf

[Definition]
actionstart = <iptables> -N f2b-<name>
              <iptables> -A f2b-<name> -j <returntype>
              <iptables> -I <chain> -p <protocol> -j f2b-<name>

actionstop = <iptables> -D <chain> -p <protocol> -j f2b-<name>
             <actionflush>
             <iptables> -X f2b-<name>

actioncheck = <iptables> -n -L <chain> | grep -q 'f2b-<name>[ \t]'

actionban = <iptables> -I f2b-<name> 1 -s <ip> -j <blocktype>
            echo "\$(date +"%%Y/%%m/%%d %%H:%%M:%%S")   BAN   [Email] = <F-USER> [IP] = <ip> banned for <bantime> seconds." >> ${iplimit_banned_log_path}

actionunban = <iptables> -D f2b-<name> -s <ip> -j <blocktype>
              echo "\$(date +"%%Y/%%m/%%d %%H:%%M:%%S")   UNBAN   [Email] = <F-USER> [IP] = <ip> unbanned." >> ${iplimit_banned_log_path}

[Init]
name = default
protocol = tcp
chain = INPUT
EOF

    echo -e "${green}Ip Limit jail files created with a bantime of ${bantime} minutes.${plain}"
}

iplimit_remove_conflicts() {
    local jail_files=(
        /etc/fail2ban/jail.conf
        /etc/fail2ban/jail.local
    )

    for file in "${jail_files[@]}"; do
        # Check for [space-ui-ipl] config in jail file then remove it
        if test -f "${file}" && grep -qw 'space-ui-ipl' ${file}; then
            sed -i "/\[space-ui-ipl\]/,/^$/d" ${file}
            echo -e "${yellow}Removing conflicts of [space-ui-ipl] in jail (${file})!${plain}\n"
        fi
    done
}

# ssh_port prints the port sshd listens on, 22 when it is not found.
ssh_port() {
    local port
    port=$(ss -ltnpH 2> /dev/null | grep '"sshd"' | awk '{print $4}' | sed 's/.*://' | sort -un | head -1)
    echo "${port:-22}"
}

# ssh_tunnel_hint prints how to reach the panel listening on $1 through an
# SSH tunnel from the user's computer.
ssh_tunnel_hint() {
    local listen="$1" port path scheme name cert sp ip opt=""
    port=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^port: .+' | awk '{print $2}')
    path=$(${space_ui_folder}/space-ui setting -show true | grep -Eo '^webBasePath: .+' | awk '{print $2}')
    name=$(site_value "website name")
    cert=$(${space_ui_folder}/space-ui setting -getCert true | sed -n 's/^cert: //p')
    scheme=http
    [[ -n "$name" || -n "$cert" ]] && scheme=https
    sp=$(ssh_port)
    [[ "$sp" != "22" ]] && opt=" -p ${sp}"
    ip=$(server_ips | head -1)
    echo -e "\n${green}On your computer:${plain}"
    echo -e "${yellow}ssh${opt} -N -L 2222:${listen}:${port} ${SUDO_USER:-root}@${ip:-<server address>}${plain}"
    echo -e "Then open: ${yellow}${scheme}://localhost:2222${path}${plain}"
    if [[ "$scheme" == "https" && -n "$name" ]]; then
        echo -e "${yellow}The certificate is for ${name}, so the browser warns about the name on localhost.${plain}"
    fi
}

# SSH_port_forwarding hides the panel's own port from the internet: the panel
# listens on a local address only and is reached through an SSH tunnel.
SSH_port_forwarding() {
    local listen choice ip
    if [[ "$(site_value "website mode")" == "reality" ]]; then
        echo -e "${yellow}The panel runs behind REALITY on 443 and has no port of its own to forward: it opens only at its secret path.${plain}"
        echo -e "${yellow}To reach it through SSH instead, put it on a port of its own first (Website & Certificate).${plain}"
        before_show_menu
        return
    fi
    listen=$(${space_ui_folder}/space-ui setting -getListen true | sed -n 's/^listenIP: //p')
    if [[ -z "$listen" || "$listen" == "0.0.0.0" || "$listen" == "::" ]]; then
        echo -e "\nThe panel listens on all addresses: its port is open to the internet."
    else
        echo -e "\n${green}The panel listens on ${listen} only.${plain}"
        ssh_tunnel_hint "$listen"
    fi

    echo -e "\nChoose an option:"
    echo -e "${green}1.${plain} Listen on a local address only (SSH forwarding)"
    echo -e "${green}2.${plain} Listen on all addresses again"
    echo -e "${green}0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice

    case "$choice" in
        1)
            echo -e "${yellow}The website and the subscriptions on the panel's port become unreachable from the internet too: clients' subscriptions stop updating.${plain}"
            read -rp "Local address to listen on [default 127.0.0.1]: " ip
            ip=${ip:-127.0.0.1}
            if ! is_ip "$ip"; then
                LOGE "\"${ip}\" is not an IP address."
                SSH_port_forwarding
                return
            fi
            ${space_ui_folder}/space-ui setting -listenIP "${ip}" > /dev/null 2>&1
            LOGI "The panel listens on ${ip} only."
            ssh_tunnel_hint "$ip"
            restart
            ;;
        2)
            ${space_ui_folder}/space-ui setting -listenIP 0.0.0.0 > /dev/null 2>&1
            LOGI "The panel listens on all addresses."
            restart
            ;;
        0)
            show_menu
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            SSH_port_forwarding
            ;;
    esac
}

show_usage() {
    echo -e "┌────────────────────────────────────────────────────────────────────┐
│  ${blue}space-ui control menu usages (subcommands):${plain}                       │
│                                                                    │
│  ${blue}space-ui${plain}                       - Admin Management Script          │
│  ${blue}space-ui start${plain}                 - Start                            │
│  ${blue}space-ui stop${plain}                  - Stop                             │
│  ${blue}space-ui restart${plain}               - Restart                          │
│  ${blue}space-ui restart-xray${plain}          - Restart Xray                     │
│  ${blue}space-ui status${plain}                - Current Status                   │
│  ${blue}space-ui settings${plain}              - Current Settings                 │
│  ${blue}space-ui site${plain}                  - Website & Certificate            │
│  ${blue}space-ui enable${plain}                - Enable Autostart on OS Startup   │
│  ${blue}space-ui disable${plain}               - Disable Autostart on OS Startup  │
│  ${blue}space-ui log${plain}                   - Check logs                       │
│  ${blue}space-ui banlog${plain}                - Check Fail2ban ban logs          │
│  ${blue}space-ui update${plain}                - Update                           │
│  ${blue}space-ui update-all-geofiles${plain}   - Update all geo files             │
│  ${blue}space-ui legacy${plain}                - Legacy version                   │
│  ${blue}space-ui install${plain}               - Install                          │
│  ${blue}space-ui uninstall${plain}             - Uninstall                        │
└────────────────────────────────────────────────────────────────────┘"
}

show_menu() {
    echo -e "
╔────────────────────────────────────────────────╗
│   ${green}Space UI Panel Management Script${plain}             │
│   ${green}0.${plain} Exit Script                               │
│────────────────────────────────────────────────│
│   ${green}1.${plain} Install                                   │
│   ${green}2.${plain} Update                                    │
│   ${green}3.${plain} Update Menu                               │
│   ${green}4.${plain} Legacy Version                            │
│   ${green}5.${plain} Uninstall                                 │
│────────────────────────────────────────────────│
│   ${green}6.${plain} Reset Username & Password                 │
│   ${green}7.${plain} Reset Web Base Path                       │
│   ${green}8.${plain} View Current Settings                     │
│────────────────────────────────────────────────│
│   ${green}9.${plain} Start                                     │
│  ${green}10.${plain} Stop                                      │
│  ${green}11.${plain} Restart                                   │
│  ${green}12.${plain} Restart Xray                              │
│  ${green}13.${plain} Check Status                              │
│  ${green}14.${plain} Logs                                      │
│────────────────────────────────────────────────│
│  ${green}15.${plain} Enable Autostart                          │
│  ${green}16.${plain} Disable Autostart                         │
│────────────────────────────────────────────────│
│  ${green}17.${plain} Website & Certificate                     │
│  ${green}18.${plain} IP Limit Management                       │
│  ${green}19.${plain} Firewall Management                       │
│  ${green}20.${plain} SSH Port Forwarding                       │
│────────────────────────────────────────────────│
│  ${green}21.${plain} Enable BBR                                │
│  ${green}22.${plain} Update Geo Files                          │
╚────────────────────────────────────────────────╝
"
    show_status
    echo && read -rp "Please enter your selection [0-22]: " num

    case "${num}" in
        0)
            exit 0
            ;;
        1)
            check_uninstall && install
            ;;
        2)
            check_install && update
            ;;
        3)
            check_install && update_menu
            ;;
        4)
            check_install && legacy_version
            ;;
        5)
            check_install && uninstall
            ;;
        6)
            check_install && reset_user
            ;;
        7)
            check_install && reset_webbasepath
            ;;
        8)
            check_install && check_config
            ;;
        9)
            check_install && start
            ;;
        10)
            check_install && stop
            ;;
        11)
            check_install && restart
            ;;
        12)
            check_install && restart_xray
            ;;
        13)
            check_install && status
            ;;
        14)
            check_install && show_log
            ;;
        15)
            check_install && enable
            ;;
        16)
            check_install && disable
            ;;
        17)
            check_install && site_menu
            ;;
        18)
            iplimit_main
            ;;
        19)
            firewall_menu
            ;;
        20)
            check_install && SSH_port_forwarding
            ;;
        21)
            bbr_menu
            ;;
        22)
            update_geo
            ;;
        *)
            LOGE "Please enter the correct number [0-22]"
            ;;
    esac
}

if [[ $# > 0 ]]; then
    case $1 in
        "start")
            check_install 0 && start 0
            ;;
        "stop")
            check_install 0 && stop 0
            ;;
        "restart")
            check_install 0 && restart 0
            ;;
        "restart-xray")
            check_install 0 && restart_xray 0
            ;;
        "status")
            check_install 0 && status 0
            ;;
        "settings")
            check_install 0 && check_config 0
            ;;
        "site")
            check_install 0 && site_menu
            ;;
        "site-setup")
            # The installer's step: the settings only, the panel starts after;
            # the REALITY client's links go to the file named, if any.
            [[ -x ${space_ui_folder}/space-ui ]] || exit 1
            site_setup || exit 1
            [[ -n "$2" && -n "$SITE_CLIENT" ]] && echo "$SITE_CLIENT" > "$2"
            exit 0
            ;;
        "site-status")
            check_install 0 && site_status "$2"
            ;;
        "enable")
            check_install 0 && enable 0
            ;;
        "disable")
            check_install 0 && disable 0
            ;;
        "log")
            check_install 0 && show_log 0
            ;;
        "banlog")
            check_install 0 && show_banlog 0
            ;;
        "update")
            check_install 0 && update 0
            ;;
        "legacy")
            check_install 0 && legacy_version 0
            ;;
        "install")
            check_uninstall 0 && install 0
            ;;
        "uninstall")
            check_install 0 && uninstall 0
            ;;
        "update-all-geofiles")
            check_install 0 && update_all_geofiles 0 && restart 0
            ;;
        *) show_usage ;;
    esac
else
    show_menu
fi
