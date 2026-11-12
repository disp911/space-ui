#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

cur_dir=$(pwd)

space_ui_folder="${SPACE_UI_MAIN_FOLDER:=/usr/local/space-ui}"
space_ui_service="${SPACE_UI_SERVICE:=/etc/systemd/system}"

# check root
[[ $EUID -ne 0 ]] && echo -e "${red}Fatal error: ${plain} Please run this script with root privilege \n " && exit 1

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
echo "The OS release is: $release"

arch() {
    case "$(uname -m)" in
        x86_64 | x64 | amd64) echo 'amd64' ;;
        i*86 | x86) echo '386' ;;
        armv8* | armv8 | arm64 | aarch64) echo 'arm64' ;;
        armv7* | armv7 | arm) echo 'armv7' ;;
        armv6* | armv6) echo 'armv6' ;;
        armv5* | armv5) echo 'armv5' ;;
        s390x) echo 's390x' ;;
        *) echo -e "${red}Unsupported CPU architecture! ${plain}" && rm -f install.sh && exit 1 ;;
    esac
}

echo "Arch: $(arch)"

# Simple helpers
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

# fetch downloads a URL to a file, over IPv4 when the default way fails.
fetch() {
    curl -fLRo "$1" "$2" || curl -4fLRo "$1" "$2"
}

# Port helpers
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

install_base() {
    case "${release}" in
        ubuntu | debian | armbian)
            apt-get update && apt-get install -y -q curl tar tzdata ca-certificates openssl
            ;;
        fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
            dnf install -y -q curl tar tzdata ca-certificates openssl
            ;;
        centos)
            if [[ "${VERSION_ID}" =~ ^7 ]]; then
                yum install -y -q curl tar tzdata ca-certificates openssl
            else
                dnf install -y -q curl tar tzdata ca-certificates openssl
            fi
            ;;
        arch | manjaro | parch)
            pacman -S --needed --noconfirm curl tar tzdata ca-certificates openssl
            ;;
        opensuse-tumbleweed | opensuse-leap)
            zypper refresh && zypper -q install -y curl tar timezone ca-certificates openssl
            ;;
        alpine)
            apk update && apk add curl tar tzdata ca-certificates openssl
            ;;
        *)
            apt-get update && apt-get install -y -q curl tar tzdata ca-certificates openssl
            ;;
    esac
}

gen_random_string() {
    local length="$1"
    openssl rand -base64 $((length * 2)) \
        | tr -dc 'a-zA-Z0-9' \
        | head -c "$length"
}

config_after_install() {
    local info=$(${space_ui_folder}/space-ui setting -show true)
    local existing_hasDefaultCredential=$(echo "$info" | grep -Eo 'hasDefaultCredential: .+' | awk '{print $2}')
    local existing_webBasePath=$(echo "$info" | grep -Eo 'webBasePath: .+' | awk '{print $2}' | sed 's#^/##')

    if [[ ${#existing_webBasePath} -lt 4 ]]; then
        existing_webBasePath=$(gen_random_string 18)
        ${space_ui_folder}/space-ui setting -webBasePath "${existing_webBasePath}" > /dev/null
        echo -e "${yellow}WebBasePath was missing or too short, a new one is set.${plain}"
    fi

    if [[ "$existing_hasDefaultCredential" == "true" ]]; then
        NEW_USERNAME=$(gen_random_string 10)
        NEW_PASSWORD=$(gen_random_string 10)
        local port
        port=$(shuf -i 1024-62000 -n 1)
        while is_port_in_use "$port"; do
            port=$(shuf -i 1024-62000 -n 1)
        done
        ${space_ui_folder}/space-ui setting -username "${NEW_USERNAME}" -password "${NEW_PASSWORD}" -port "${port}" > /dev/null

        echo ""
        echo -e "${green}═══════════════════════════════════════════${plain}"
        echo -e "${green}     Website, Panel Address & Certificate  ${plain}"
        echo -e "${green}═══════════════════════════════════════════${plain}"
        echo -e "${yellow}The panel shows a website of its own at the root and opens at a secret path.${plain}"
        echo -e "${yellow}It obtains and renews its certificate itself.${plain}"
        echo ""
        SITE_CLIENT_FILE=$(mktemp)
        if ! /usr/bin/space-ui site-setup "${SITE_CLIENT_FILE}"; then
            echo -e "${yellow}The website is not set up; the panel opens on its own port over plain HTTP. Set it up later: space-ui site${plain}"
        fi
    fi

    ${space_ui_folder}/space-ui migrate
}

# show_access prints how to reach the panel once it runs: its address, and on
# a new installation the login and REALITY's first client.
show_access() {
    echo ""
    echo -e "${green}═══════════════════════════════════════════${plain}"
    echo -e "${green}     Panel Installation Complete!         ${plain}"
    echo -e "${green}═══════════════════════════════════════════${plain}"
    /usr/bin/space-ui site-status wait 2> /dev/null
    if [[ -n "${NEW_USERNAME}" ]]; then
        echo -e "${green}Username:    ${NEW_USERNAME}${plain}"
        echo -e "${green}Password:    ${NEW_PASSWORD}${plain}"
    fi
    if [[ -s "${SITE_CLIENT_FILE}" ]]; then
        echo -e "${green}REALITY on 443, first client:${plain}"
        cat "${SITE_CLIENT_FILE}"
    fi
    rm -f "${SITE_CLIENT_FILE}"
    echo -e "${green}═══════════════════════════════════════════${plain}"
    if [[ -n "${NEW_USERNAME}" ]]; then
        echo -e "${yellow}⚠ IMPORTANT: Save these credentials securely!${plain}"
    fi
}

install_space-ui() {
    cd ${space_ui_folder%/space-ui}/

    # Download resources; SPACE_UI_ARCHIVE names an archive already on the server
    if [[ -n "${SPACE_UI_ARCHIVE}" ]]; then
        tag_version="(local archive)"
        cp -f "${SPACE_UI_ARCHIVE}" ${space_ui_folder}-linux-$(arch).tar.gz || exit 1
    elif [ $# == 0 ]; then
        tag_version=$(curl -Ls "https://api.github.com/repos/disp911/space-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$tag_version" ]]; then
            echo -e "${yellow}Trying to fetch version with IPv4...${plain}"
            tag_version=$(curl -4 -Ls "https://api.github.com/repos/disp911/space-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
            if [[ ! -n "$tag_version" ]]; then
                echo -e "${red}Failed to fetch space-ui version, it may be due to GitHub API restrictions, please try it later${plain}"
                exit 1
            fi
        fi
        echo -e "Got space-ui latest version: ${tag_version}, beginning the installation..."
        fetch ${space_ui_folder}-linux-$(arch).tar.gz https://github.com/disp911/space-ui/releases/download/${tag_version}/space-ui-linux-$(arch).tar.gz
        if [[ $? -ne 0 ]]; then
            echo -e "${red}Downloading space-ui failed, please be sure that your server can access GitHub ${plain}"
            exit 1
        fi
    else
        tag_version=$1
        tag_version_numeric=${tag_version#v}
        min_version="3.0.0"

        if [[ "$(printf '%s\n' "$min_version" "$tag_version_numeric" | sort -V | head -n1)" != "$min_version" ]]; then
            echo -e "${red}Please use a newer version (at least v3.0.0). Exiting installation.${plain}"
            exit 1
        fi

        url="https://github.com/disp911/space-ui/releases/download/${tag_version}/space-ui-linux-$(arch).tar.gz"
        echo -e "Beginning to install space-ui $1"
        fetch ${space_ui_folder}-linux-$(arch).tar.gz ${url}
        if [[ $? -ne 0 ]]; then
            echo -e "${red}Download space-ui $1 failed, please check if the version exists ${plain}"
            exit 1
        fi
    fi
    # Stop space-ui service and remove old resources
    if [[ -e ${space_ui_folder}/ ]]; then
        if [[ $release == "alpine" ]]; then
            rc-service space-ui stop
        else
            systemctl stop space-ui
        fi
        rm ${space_ui_folder}/ -rf
    fi

    # Extract resources and set permissions
    tar zxf space-ui-linux-$(arch).tar.gz
    rm space-ui-linux-$(arch).tar.gz -f

    cd space-ui
    chmod +x space-ui
    chmod +x space-ui.sh

    # Check the system's architecture and rename the file accordingly
    if [[ $(arch) == "armv5" || $(arch) == "armv6" || $(arch) == "armv7" ]]; then
        mv bin/xray-linux-$(arch) bin/xray-linux-arm
        chmod +x bin/xray-linux-arm
    fi
    chmod +x space-ui bin/xray-linux-$(arch)
    chmod +x bin/telemt-linux-$(arch) 2> /dev/null

    # The space-ui menu of the same release
    cp -f space-ui.sh /usr/bin/space-ui
    chmod +x /usr/bin/space-ui
    mkdir -p /var/log/space-ui
    config_after_install

    # Etckeeper compatibility
    if [ -d "/etc/.git" ]; then
        if [ -f "/etc/.gitignore" ]; then
            if ! grep -q "space-ui/space-ui.db" "/etc/.gitignore"; then
                echo "" >> "/etc/.gitignore"
                echo "space-ui/space-ui.db" >> "/etc/.gitignore"
                echo -e "${green}Added space-ui.db to /etc/.gitignore for etckeeper${plain}"
            fi
        else
            echo "space-ui/space-ui.db" > "/etc/.gitignore"
            echo -e "${green}Created /etc/.gitignore and added space-ui.db for etckeeper${plain}"
        fi
    fi

    # Startup files the archive lacks come from the same release, or from main
    # for a local archive.
    local ref=main
    [[ -z "${SPACE_UI_ARCHIVE}" ]] && ref=${tag_version}

    if [[ $release == "alpine" ]]; then
        fetch /etc/init.d/space-ui https://raw.githubusercontent.com/disp911/space-ui/${ref}/space-ui.rc
        if [[ $? -ne 0 ]]; then
            echo -e "${red}Failed to download space-ui.rc${plain}"
            exit 1
        fi
        chmod +x /etc/init.d/space-ui
        rc-update add space-ui
        rc-service space-ui start
    else
        # Install systemd service file
        service_installed=false

        if [ -f "space-ui.service" ]; then
            echo -e "${green}Found space-ui.service in extracted files, installing...${plain}"
            cp -f space-ui.service ${space_ui_service}/ > /dev/null 2>&1
            if [[ $? -eq 0 ]]; then
                service_installed=true
            fi
        fi

        if [ "$service_installed" = false ]; then
            case "${release}" in
                ubuntu | debian | armbian)
                    if [ -f "space-ui.service.debian" ]; then
                        echo -e "${green}Found space-ui.service.debian in extracted files, installing...${plain}"
                        cp -f space-ui.service.debian ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
                arch | manjaro | parch)
                    if [ -f "space-ui.service.arch" ]; then
                        echo -e "${green}Found space-ui.service.arch in extracted files, installing...${plain}"
                        cp -f space-ui.service.arch ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
                *)
                    if [ -f "space-ui.service.rhel" ]; then
                        echo -e "${green}Found space-ui.service.rhel in extracted files, installing...${plain}"
                        cp -f space-ui.service.rhel ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
            esac
        fi

        # If service file not found in tar.gz, download from GitHub
        if [ "$service_installed" = false ]; then
            echo -e "${yellow}Service files not found in tar.gz, downloading from GitHub...${plain}"
            case "${release}" in
                ubuntu | debian | armbian)
                    fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${ref}/space-ui.service.debian > /dev/null 2>&1
                    ;;
                arch | manjaro | parch)
                    fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${ref}/space-ui.service.arch > /dev/null 2>&1
                    ;;
                *)
                    fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${ref}/space-ui.service.rhel > /dev/null 2>&1
                    ;;
            esac

            if [[ $? -ne 0 ]]; then
                echo -e "${red}Failed to install space-ui.service from GitHub${plain}"
                exit 1
            fi
            service_installed=true
        fi

        if [ "$service_installed" = true ]; then
            echo -e "${green}Setting up systemd unit...${plain}"
            chown root:root ${space_ui_service}/space-ui.service > /dev/null 2>&1
            chmod 644 ${space_ui_service}/space-ui.service > /dev/null 2>&1
            systemctl daemon-reload
            systemctl enable space-ui
            systemctl start space-ui
        else
            echo -e "${red}Failed to install space-ui.service file${plain}"
            exit 1
        fi
    fi

    echo -e "${green}space-ui ${tag_version}${plain} installation finished, it is running now..."
    /usr/bin/space-ui help
}

echo -e "${green}Running...${plain}"
install_base
install_space-ui $1
show_access
