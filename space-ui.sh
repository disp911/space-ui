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
[[ "$1" == site-* ]] || echo "The OS release is: $release"

os_version=""
os_version=$(grep "^VERSION_ID" /etc/os-release | cut -d '=' -f2 | tr -d '"' | tr -d '.')

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

update_menu() {
    echo -e "${yellow}Updating Menu${plain}"
    confirm "This function will update the menu to the latest changes." "y"
    if [[ $? != 0 ]]; then
        LOGE "Cancelled"
        if [[ $# == 0 ]]; then
            before_show_menu
        fi
        return 0
    fi

    curl -fLRo /usr/bin/space-ui https://raw.githubusercontent.com/disp911/space-ui/main/space-ui.sh
    chmod +x ${space_ui_folder}/space-ui.sh
    chmod +x /usr/bin/space-ui

    if [[ $? == 0 ]]; then
        echo -e "${green}Update successful. The panel has automatically restarted.${plain}"
        exit 0
    else
        echo -e "${red}Failed to update the menu.${plain}"
        return 1
    fi
}

legacy_version() {
    echo -n "Enter the panel version (like 2.9.5):"
    read -r tag_version

    if [ -z "$tag_version" ]; then
        echo "Panel version cannot be empty. Exiting."
        exit 1
    fi
    # Use the entered panel version in the download link
    install_command="bash <(curl -Ls "https://raw.githubusercontent.com/disp911/space-ui/v$tag_version/install.sh") v$tag_version"

    echo "Downloading and installing panel version $tag_version..."
    eval $install_command
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

reset_config() {
    confirm "Are you sure you want to reset all panel settings, Account data will not be lost, Username and password will not change" "n"
    if [[ $? != 0 ]]; then
        if [[ $# == 0 ]]; then
            show_menu
        fi
        return 0
    fi
    ${space_ui_folder}/space-ui setting -reset
    echo -e "All panel settings have been reset to default."
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

set_port() {
    echo -n "Enter port number[1-65535]: "
    read -r port
    if [[ -z "${port}" ]]; then
        LOGD "Cancelled"
        before_show_menu
    else
        ${space_ui_folder}/space-ui setting -port ${port}
        echo -e "The port is set, Please restart the panel now, and use the new port ${green}${port}${plain} to access web panel"
        confirm_restart
    fi
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
        echo -e "${green}\t1.${plain} Debug Log"
        echo -e "${green}\t0.${plain} Back to Main Menu"
        read -rp "Choose an option: " choice

        case "$choice" in
            0)
                show_menu
                ;;
            1)
                grep -F 'space-ui[' /var/log/messages
                if [[ $# == 0 ]]; then
                    before_show_menu
                fi
                ;;
            *)
                echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
                show_log
                ;;
        esac
    else
        echo -e "${green}\t1.${plain} Debug Log"
        echo -e "${green}\t2.${plain} Clear All logs"
        echo -e "${green}\t0.${plain} Back to Main Menu"
        read -rp "Choose an option: " choice

        case "$choice" in
            0)
                show_menu
                ;;
            1)
                journalctl -u space-ui -e --no-pager -f -p debug
                if [[ $# == 0 ]]; then
                    before_show_menu
                fi
                ;;
            2)
                sudo journalctl --rotate
                sudo journalctl --vacuum-time=1s
                echo "All Logs cleared."
                restart
                ;;
            *)
                echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
                show_log
                ;;
        esac
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

update_shell() {
    curl -fLRo /usr/bin/space-ui -z /usr/bin/space-ui https://github.com/disp911/space-ui/raw/main/space-ui.sh
    if [[ $? != 0 ]]; then
        echo ""
        LOGE "Failed to download script, Please check whether the machine can connect Github"
        before_show_menu
    else
        chmod +x /usr/bin/space-ui
        LOGI "Upgrade script succeeded, Please rerun the script"
        before_show_menu
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

firewall_menu() {
    echo -e "${green}\t1.${plain} ${green}Install${plain} Firewall"
    echo -e "${green}\t2.${plain} Port List [numbered]"
    echo -e "${green}\t3.${plain} ${green}Open${plain} Ports"
    echo -e "${green}\t4.${plain} ${red}Delete${plain} Ports from List"
    echo -e "${green}\t5.${plain} ${green}Enable${plain} Firewall"
    echo -e "${green}\t6.${plain} ${red}Disable${plain} Firewall"
    echo -e "${green}\t7.${plain} Firewall Status"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " choice
    case "$choice" in
        0)
            show_menu
            ;;
        1)
            install_firewall
            firewall_menu
            ;;
        2)
            ufw status numbered
            firewall_menu
            ;;
        3)
            open_ports
            firewall_menu
            ;;
        4)
            delete_ports
            firewall_menu
            ;;
        5)
            ufw enable
            firewall_menu
            ;;
        6)
            ufw disable
            firewall_menu
            ;;
        7)
            ufw status verbose
            firewall_menu
            ;;
        *)
            echo -e "${red}Invalid option. Please select a valid number.${plain}\n"
            firewall_menu
            ;;
    esac
}

install_firewall() {
    if ! command -v ufw &> /dev/null; then
        echo "ufw firewall is not installed. Installing now..."
        apt-get update
        apt-get install -y ufw
    else
        echo "ufw firewall is already installed"
    fi

    # Check if the firewall is inactive
    if ufw status | grep -q "Status: active"; then
        echo "Firewall is already active"
    else
        echo "Activating firewall..."
        # Open the necessary ports
        ufw allow ssh
        ufw allow http
        ufw allow https
        ufw allow 2053/tcp #webPort
        ufw allow 2096/tcp #subport

        # Enable the firewall
        ufw --force enable
    fi
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
    update_geofiles "IR"
    update_geofiles "RU"
}

update_geofiles() {
    case "${1}" in
        "main")
            dat_files=(geoip geosite)
            dat_source="Loyalsoldier/v2ray-rules-dat"
            ;;
        "IR")
            dat_files=(geoip_IR geosite_IR)
            dat_source="chocolate4u/Iran-v2ray-rules"
            ;;
        "RU")
            dat_files=(geoip_RU geosite_RU)
            dat_source="runetfreedom/russia-v2ray-rules-dat"
            ;;
    esac
    for dat in "${dat_files[@]}"; do
        # Remove suffix for remote filename (e.g., geoip_IR -> geoip)
        remote_file="${dat%%_*}"
        curl -fLRo ${space_ui_folder}/bin/${dat}.dat -z ${space_ui_folder}/bin/${dat}.dat \
            https://github.com/${dat_source}/releases/latest/download/${remote_file}.dat
    done
}

update_geo() {
    echo -e "${green}\t1.${plain} Loyalsoldier (geoip.dat, geosite.dat)"
    echo -e "${green}\t2.${plain} chocolate4u (geoip_IR.dat, geosite_IR.dat)"
    echo -e "${green}\t3.${plain} runetfreedom (geoip_RU.dat, geosite_RU.dat)"
    echo -e "${green}\t4.${plain} All"
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
            update_geofiles "IR"
            echo -e "${green}chocolate4u datasets have been updated successfully!${plain}"
            restart
            ;;
        3)
            update_geofiles "RU"
            echo -e "${green}runetfreedom datasets have been updated successfully!${plain}"
            restart
            ;;
        4)
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

run_speedtest() {
    # Check if Speedtest is already installed
    if ! command -v speedtest &> /dev/null; then
        # If not installed, determine installation method
        if command -v snap &> /dev/null; then
            # Use snap to install Speedtest
            echo "Installing Speedtest using snap..."
            snap install speedtest
        else
            # Fallback to using package managers
            local pkg_manager=""
            local speedtest_install_script=""

            if command -v dnf &> /dev/null; then
                pkg_manager="dnf"
                speedtest_install_script="https://packagecloud.io/install/repositories/ookla/speedtest-cli/script.rpm.sh"
            elif command -v yum &> /dev/null; then
                pkg_manager="yum"
                speedtest_install_script="https://packagecloud.io/install/repositories/ookla/speedtest-cli/script.rpm.sh"
            elif command -v apt-get &> /dev/null; then
                pkg_manager="apt-get"
                speedtest_install_script="https://packagecloud.io/install/repositories/ookla/speedtest-cli/script.deb.sh"
            elif command -v apt &> /dev/null; then
                pkg_manager="apt"
                speedtest_install_script="https://packagecloud.io/install/repositories/ookla/speedtest-cli/script.deb.sh"
            fi

            if [[ -z $pkg_manager ]]; then
                echo "Error: Package manager not found. You may need to install Speedtest manually."
                return 1
            else
                echo "Installing Speedtest using $pkg_manager..."
                curl -s $speedtest_install_script | bash
                $pkg_manager install -y speedtest
            fi
        fi
    fi

    speedtest
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
    echo -e "${green}\t10.${plain} Uninstall Fail2ban and IP Limit"
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

        # Install fail2ban together with nftables. Recent fail2ban packages
        # default to `banaction = nftables-multiport` in /etc/fail2ban/jail.conf,
        # but the `nftables` package isn't pulled in as a dependency on most
        # minimal server images (Debian 12+, Ubuntu 24+, fresh RHEL-family).
        # Without `nft` in PATH the default sshd jail fails to ban with
        #   stderr: '/bin/sh: 1: nft: not found'
        # even though our own space-ui-ipl jail uses iptables. Bundling the binary
        # at install time prevents that confusing log spam for new installs.
        case "${release}" in
            ubuntu)
                apt-get update
                if [[ "${os_version}" -ge 24 ]]; then
                    apt-get install python3-pip -y
                    python3 -m pip install pyasynchat --break-system-packages
                fi
                apt-get install fail2ban nftables -y
                ;;
            debian)
                apt-get update
                if [ "$os_version" -ge 12 ]; then
                    apt-get install -y python3-systemd
                fi
                apt-get install -y fail2ban nftables
                ;;
            armbian)
                apt-get update && apt-get install fail2ban nftables -y
                ;;
            fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
                dnf -y update && dnf -y install fail2ban nftables
                ;;
            centos)
                if [[ "${VERSION_ID}" =~ ^7 ]]; then
                    yum update -y && yum install epel-release -y
                    yum -y install fail2ban nftables
                else
                    dnf -y update && dnf -y install fail2ban nftables
                fi
                ;;
            arch | manjaro | parch)
                pacman -Syu --noconfirm fail2ban nftables
                ;;
            alpine)
                apk add fail2ban nftables
                ;;
            *)
                echo -e "${red}Unsupported operating system. Please check the script and install the necessary packages manually.${plain}\n"
                exit 1
                ;;
        esac

        if ! command -v fail2ban-client &> /dev/null; then
            echo -e "${red}Fail2ban installation failed.${plain}\n"
            exit 1
        fi

        echo -e "${green}Fail2ban installed successfully!${plain}\n"
    else
        echo -e "${yellow}Fail2ban is already installed.${plain}\n"
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
        if ! systemctl is-active --quiet fail2ban; then
            systemctl start fail2ban
        else
            systemctl restart fail2ban
        fi
        systemctl enable fail2ban
    fi

    echo -e "${green}IP Limit installed and configured successfully!${plain}\n"
    before_show_menu
}

remove_iplimit() {
    echo -e "${green}\t1.${plain} Only remove IP Limit configurations"
    echo -e "${green}\t2.${plain} Uninstall Fail2ban and IP Limit"
    echo -e "${green}\t0.${plain} Back to Main Menu"
    read -rp "Choose an option: " num
    case "$num" in
        1)
            rm -f /etc/fail2ban/filter.d/space-ui-ipl.conf
            rm -f /etc/fail2ban/action.d/space-ui-ipl.conf
            rm -f /etc/fail2ban/jail.d/space-ui-ipl.conf
            if [[ $release == "alpine" ]]; then
                rc-service fail2ban restart
            else
                systemctl restart fail2ban
            fi
            echo -e "${green}IP Limit removed successfully!${plain}\n"
            before_show_menu
            ;;
        2)
            rm -rf /etc/fail2ban
            if [[ $release == "alpine" ]]; then
                rc-service fail2ban stop
            else
                systemctl stop fail2ban
            fi
            case "${release}" in
                ubuntu | debian | armbian)
                    apt-get remove -y fail2ban
                    apt-get purge -y fail2ban -y
                    apt-get autoremove -y
                    ;;
                fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
                    dnf remove fail2ban -y
                    dnf autoremove -y
                    ;;
                centos)
                    if [[ "${VERSION_ID}" =~ ^7 ]]; then
                        yum remove fail2ban -y
                        yum autoremove -y
                    else
                        dnf remove fail2ban -y
                        dnf autoremove -y
                    fi
                    ;;
                arch | manjaro | parch)
                    pacman -Rns --noconfirm fail2ban
                    ;;
                alpine)
                    apk del fail2ban
                    ;;
                *)
                    echo -e "${red}Unsupported operating system. Please uninstall Fail2ban manually.${plain}\n"
                    exit 1
                    ;;
            esac
            echo -e "${green}Fail2ban and IP Limit removed successfully!${plain}\n"
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

SSH_port_forwarding() {
    local URL_lists=(
        "https://api4.ipify.org"
        "https://ipv4.icanhazip.com"
        "https://v4.api.ipinfo.io/ip"
        "https://ipv4.myexternalip.com/raw"
        "https://4.ident.me"
        "https://check-host.net/ip"
    )
    local server_ip=""
    for ip_address in "${URL_lists[@]}"; do
        local response=$(curl -s -w "\n%{http_code}" --max-time 3 "${ip_address}" 2> /dev/null)
        local http_code=$(echo "$response" | tail -n1)
        local ip_result=$(echo "$response" | head -n-1 | tr -d '[:space:]')
        if [[ "${http_code}" == "200" && -n "${ip_result}" ]]; then
            server_ip="${ip_result}"
            break
        fi
    done

    local existing_webBasePath=$(${space_ui_folder}/space-ui setting -show true | grep -Eo 'webBasePath: .+' | awk '{print $2}')
    local existing_port=$(${space_ui_folder}/space-ui setting -show true | grep -Eo 'port: .+' | awk '{print $2}')
    local existing_listenIP=$(${space_ui_folder}/space-ui setting -getListen true | grep -Eo 'listenIP: .+' | awk '{print $2}')
    local existing_cert=$(${space_ui_folder}/space-ui setting -getCert true | grep -Eo 'cert: .+' | awk '{print $2}')
    local existing_key=$(${space_ui_folder}/space-ui setting -getCert true | grep -Eo 'key: .+' | awk '{print $2}')

    local config_listenIP=""
    local listen_choice=""

    if [[ -n "$existing_cert" && -n "$existing_key" ]]; then
        echo -e "${green}Panel is secure with SSL.${plain}"
        before_show_menu
    fi
    if [[ -z "$existing_cert" && -z "$existing_key" && (-z "$existing_listenIP" || "$existing_listenIP" == "0.0.0.0") ]]; then
        echo -e "\n${red}Warning: No Cert and Key found! The panel is not secure.${plain}"
        echo "Please obtain a certificate or set up SSH port forwarding."
    fi

    if [[ -n "$existing_listenIP" && "$existing_listenIP" != "0.0.0.0" && (-z "$existing_cert" && -z "$existing_key") ]]; then
        echo -e "\n${green}Current SSH Port Forwarding Configuration:${plain}"
        echo -e "Standard SSH command:"
        echo -e "${yellow}ssh -L 2222:${existing_listenIP}:${existing_port} root@${server_ip}${plain}"
        echo -e "\nIf using SSH key:"
        echo -e "${yellow}ssh -i <sshkeypath> -L 2222:${existing_listenIP}:${existing_port} root@${server_ip}${plain}"
        echo -e "\nAfter connecting, access the panel at:"
        echo -e "${yellow}http://localhost:2222${existing_webBasePath}${plain}"
    fi

    echo -e "\nChoose an option:"
    echo -e "${green}1.${plain} Set listen IP"
    echo -e "${green}2.${plain} Clear listen IP"
    echo -e "${green}0.${plain} Back to Main Menu"
    read -rp "Choose an option: " num

    case "$num" in
        1)
            if [[ -z "$existing_listenIP" || "$existing_listenIP" == "0.0.0.0" ]]; then
                echo -e "\nNo listenIP configured. Choose an option:"
                echo -e "1. Use default IP (127.0.0.1)"
                echo -e "2. Set a custom IP"
                read -rp "Select an option (1 or 2): " listen_choice

                config_listenIP="127.0.0.1"
                [[ "$listen_choice" == "2" ]] && read -rp "Enter custom IP to listen on: " config_listenIP

                ${space_ui_folder}/space-ui setting -listenIP "${config_listenIP}" > /dev/null 2>&1
                echo -e "${green}listen IP has been set to ${config_listenIP}.${plain}"
                echo -e "\n${green}SSH Port Forwarding Configuration:${plain}"
                echo -e "Standard SSH command:"
                echo -e "${yellow}ssh -L 2222:${config_listenIP}:${existing_port} root@${server_ip}${plain}"
                echo -e "\nIf using SSH key:"
                echo -e "${yellow}ssh -i <sshkeypath> -L 2222:${config_listenIP}:${existing_port} root@${server_ip}${plain}"
                echo -e "\nAfter connecting, access the panel at:"
                echo -e "${yellow}http://localhost:2222${existing_webBasePath}${plain}"
                restart
            else
                config_listenIP="${existing_listenIP}"
                echo -e "${green}Current listen IP is already set to ${config_listenIP}.${plain}"
            fi
            ;;
        2)
            ${space_ui_folder}/space-ui setting -listenIP 0.0.0.0 > /dev/null 2>&1
            echo -e "${green}Listen IP has been cleared.${plain}"
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
│   ${green}8.${plain} Reset Settings                            │
│   ${green}9.${plain} Change Port                               │
│  ${green}10.${plain} View Current Settings                     │
│────────────────────────────────────────────────│
│  ${green}11.${plain} Start                                     │
│  ${green}12.${plain} Stop                                      │
│  ${green}13.${plain} Restart                                   │
|  ${green}14.${plain} Restart Xray                              │
│  ${green}15.${plain} Check Status                              │
│  ${green}16.${plain} Logs Management                           │
│────────────────────────────────────────────────│
│  ${green}17.${plain} Enable Autostart                          │
│  ${green}18.${plain} Disable Autostart                         │
│────────────────────────────────────────────────│
│  ${green}19.${plain} Website & Certificate                     │
│  ${green}20.${plain} IP Limit Management                       │
│  ${green}21.${plain} Firewall Management                       │
│  ${green}22.${plain} SSH Port Forwarding Management            │
│────────────────────────────────────────────────│
│  ${green}23.${plain} Enable BBR                                │
│  ${green}24.${plain} Update Geo Files                          │
│  ${green}25.${plain} Speedtest by Ookla                        │
╚────────────────────────────────────────────────╝
"
    show_status
    echo && read -rp "Please enter your selection [0-25]: " num

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
            check_install && reset_config
            ;;
        9)
            check_install && set_port
            ;;
        10)
            check_install && check_config
            ;;
        11)
            check_install && start
            ;;
        12)
            check_install && stop
            ;;
        13)
            check_install && restart
            ;;
        14)
            check_install && restart_xray
            ;;
        15)
            check_install && status
            ;;
        16)
            check_install && show_log
            ;;
        17)
            check_install && enable
            ;;
        18)
            check_install && disable
            ;;
        19)
            check_install && site_menu
            ;;
        20)
            iplimit_main
            ;;
        21)
            firewall_menu
            ;;
        22)
            SSH_port_forwarding
            ;;
        23)
            bbr_menu
            ;;
        24)
            update_geo
            ;;
        25)
            run_speedtest
            ;;
        *)
            LOGE "Please enter the correct number [0-25]"
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
