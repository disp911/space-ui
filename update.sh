#!/bin/bash

red='\033[0;31m'
green='\033[0;32m'
blue='\033[0;34m'
yellow='\033[0;33m'
plain='\033[0m'

space_ui_folder="${SPACE_UI_MAIN_FOLDER:=/usr/local/space-ui}"
space_ui_service="${SPACE_UI_SERVICE:=/etc/systemd/system}"

# Don't edit this config
b_source="${BASH_SOURCE[0]}"
while [ -h "$b_source" ]; do
    b_dir="$(cd -P "$(dirname "$b_source")" > /dev/null 2>&1 && pwd || pwd -P)"
    b_source="$(readlink "$b_source")"
    [[ $b_source != /* ]] && b_source="$b_dir/$b_source"
done
cur_dir="$(cd -P "$(dirname "$b_source")" > /dev/null 2>&1 && pwd || pwd -P)"
script_name=$(basename "$0")

# Check command exist function
_command_exists() {
    type "$1" &> /dev/null
}

# Fail, log and exit script function
_fail() {
    local msg=${1}
    echo -e "${red}${msg}${plain}"
    exit 2
}

# check root
[[ $EUID -ne 0 ]] && _fail "FATAL ERROR: Please run this script with root privilege."

if _command_exists curl; then
    curl_bin=$(which curl)
else
    _fail "ERROR: Command 'curl' not found."
fi

# Check OS and set release variable
if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    release=$ID
elif [[ -f /usr/lib/os-release ]]; then
    source /usr/lib/os-release
    release=$ID
else
    _fail "Failed to check the system OS, please contact the author!"
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
        *) echo -e "${red}Unsupported CPU architecture!${plain}" && rm -f "${cur_dir}/${script_name}" > /dev/null 2>&1 && exit 2 ;;
    esac
}

echo "Arch: $(arch)"

# fetch downloads a URL to a file, over IPv4 when the default way fails.
fetch() {
    ${curl_bin} -fLRo "$1" "$2" > /dev/null 2>&1 || ${curl_bin} -4fLRo "$1" "$2" > /dev/null 2>&1
}

# Port helpers

gen_random_string() {
    local length="$1"
    openssl rand -base64 $((length * 2)) \
        | tr -dc 'a-zA-Z0-9' \
        | head -c "$length"
}

install_base() {
    echo -e "${green}Installing dependency packages...${plain}"
    case "${release}" in
        ubuntu | debian | armbian)
            apt-get update > /dev/null 2>&1 && apt-get install -y -q curl tar tzdata openssl > /dev/null 2>&1
            ;;
        fedora | amzn | virtuozzo | rhel | almalinux | rocky | ol)
            dnf install -y -q curl tar tzdata openssl > /dev/null 2>&1
            ;;
        centos)
            if [[ "${VERSION_ID}" =~ ^7 ]]; then
                yum install -y -q curl tar tzdata openssl > /dev/null 2>&1
            else
                dnf install -y -q curl tar tzdata openssl > /dev/null 2>&1
            fi
            ;;
        arch | manjaro | parch)
            pacman -S --needed --noconfirm curl tar tzdata openssl > /dev/null 2>&1
            ;;
        opensuse-tumbleweed | opensuse-leap)
            zypper refresh > /dev/null 2>&1 && zypper -q install -y curl tar timezone openssl > /dev/null 2>&1
            ;;
        alpine)
            apk update > /dev/null 2>&1 && apk add curl tar tzdata openssl > /dev/null 2>&1
            ;;
        *)
            apt-get update > /dev/null 2>&1 && apt install -y -q curl tar tzdata openssl > /dev/null 2>&1
            ;;
    esac
}

config_after_update() {
    ${space_ui_folder}/space-ui migrate

    local existing_webBasePath=$(${space_ui_folder}/space-ui setting -show true | grep -Eo 'webBasePath: .+' | awk '{print $2}' | sed 's#^/##')
    if [[ ${#existing_webBasePath} -lt 4 ]]; then
        echo -e "${yellow}WebBasePath is missing or too short. Generating a new one...${plain}"
        ${space_ui_folder}/space-ui setting -webBasePath "$(gen_random_string 18)" > /dev/null
    fi
}

update_space-ui() {
    cd ${space_ui_folder%/space-ui}/

    if [ -f "${space_ui_folder}/space-ui" ]; then
        current_version=$(${space_ui_folder}/space-ui -v)
        echo -e "${green}Current space-ui version: ${current_version}${plain}"
    else
        _fail "ERROR: Current space-ui version: unknown"
    fi

    echo -e "${green}Downloading new space-ui version...${plain}"

    tag_version=$(${curl_bin} -Ls "https://api.github.com/repos/disp911/space-ui/releases/latest" 2> /dev/null | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ ! -n "$tag_version" ]]; then
        echo -e "${yellow}Trying to fetch version with IPv4...${plain}"
        tag_version=$(${curl_bin} -4 -Ls "https://api.github.com/repos/disp911/space-ui/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
        if [[ ! -n "$tag_version" ]]; then
            _fail "ERROR: Failed to fetch space-ui version, it may be due to GitHub API restrictions, please try it later"
        fi
    fi
    echo -e "Got space-ui latest version: ${tag_version}, beginning the installation..."
    ${curl_bin} -fLRo ${space_ui_folder}-linux-$(arch).tar.gz https://github.com/disp911/space-ui/releases/download/${tag_version}/space-ui-linux-$(arch).tar.gz 2> /dev/null
    if [[ $? -ne 0 ]]; then
        echo -e "${yellow}Trying to fetch version with IPv4...${plain}"
        ${curl_bin} -4fLRo ${space_ui_folder}-linux-$(arch).tar.gz https://github.com/disp911/space-ui/releases/download/${tag_version}/space-ui-linux-$(arch).tar.gz 2> /dev/null
        if [[ $? -ne 0 ]]; then
            _fail "ERROR: Failed to download space-ui, please be sure that your server can access GitHub"
        fi
    fi

    if [[ -e ${space_ui_folder}/ ]]; then
        echo -e "${green}Stopping space-ui...${plain}"
        if [[ $release == "alpine" ]]; then
            if [ -f "/etc/init.d/space-ui" ]; then
                rc-service space-ui stop > /dev/null 2>&1
                rc-update del space-ui > /dev/null 2>&1
                echo -e "${green}Removing old service unit version...${plain}"
                rm -f /etc/init.d/space-ui > /dev/null 2>&1
            else
                rm space-ui-linux-$(arch).tar.gz -f > /dev/null 2>&1
                _fail "ERROR: space-ui service unit not installed."
            fi
        else
            if [ -f "${space_ui_service}/space-ui.service" ]; then
                systemctl stop space-ui > /dev/null 2>&1
                systemctl disable space-ui > /dev/null 2>&1
                echo -e "${green}Removing old systemd unit version...${plain}"
                rm ${space_ui_service}/space-ui.service -f > /dev/null 2>&1
                systemctl daemon-reload > /dev/null 2>&1
            else
                rm space-ui-linux-$(arch).tar.gz -f > /dev/null 2>&1
                _fail "ERROR: space-ui systemd unit not installed."
            fi
        fi
        echo -e "${green}Removing old space-ui version...${plain}"
        rm ${space_ui_folder} -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui.service -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui.service.debian -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui.service.arch -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui.service.rhel -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui -f > /dev/null 2>&1
        rm ${space_ui_folder}/space-ui.sh -f > /dev/null 2>&1
        echo -e "${green}Removing old xray version...${plain}"
        rm ${space_ui_folder}/bin/xray-linux-amd64 -f > /dev/null 2>&1
        echo -e "${green}Removing old README and LICENSE file...${plain}"
        rm ${space_ui_folder}/bin/README.md -f > /dev/null 2>&1
        rm ${space_ui_folder}/bin/LICENSE -f > /dev/null 2>&1
    else
        rm space-ui-linux-$(arch).tar.gz -f > /dev/null 2>&1
        _fail "ERROR: space-ui not installed. Versions before 3.0.0 are not updated: install Space UI anew with install.sh."
    fi

    echo -e "${green}Installing new space-ui version...${plain}"
    tar zxf space-ui-linux-$(arch).tar.gz > /dev/null 2>&1
    rm space-ui-linux-$(arch).tar.gz -f > /dev/null 2>&1
    cd space-ui > /dev/null 2>&1
    chmod +x space-ui > /dev/null 2>&1

    # Check the system's architecture and rename the file accordingly
    if [[ $(arch) == "armv5" || $(arch) == "armv6" || $(arch) == "armv7" ]]; then
        mv bin/xray-linux-$(arch) bin/xray-linux-arm > /dev/null 2>&1
        chmod +x bin/xray-linux-arm > /dev/null 2>&1
    fi

    chmod +x space-ui bin/xray-linux-$(arch) > /dev/null 2>&1
    chmod +x bin/telemt-linux-$(arch) > /dev/null 2>&1

    echo -e "${green}Installing the space-ui menu of the same release...${plain}"
    cp -f space-ui.sh /usr/bin/space-ui > /dev/null 2>&1 || _fail "ERROR: space-ui.sh is missing from the release"

    chmod +x ${space_ui_folder}/space-ui.sh > /dev/null 2>&1
    chmod +x /usr/bin/space-ui > /dev/null 2>&1
    mkdir -p /var/log/space-ui > /dev/null 2>&1

    echo -e "${green}Changing owner...${plain}"
    chown -R root:root ${space_ui_folder} > /dev/null 2>&1

    if [ -f "${space_ui_folder}/bin/config.json" ]; then
        echo -e "${green}Changing on config file permissions...${plain}"
        chmod 640 ${space_ui_folder}/bin/config.json > /dev/null 2>&1
    fi

    if [[ $release == "alpine" ]]; then
        echo -e "${green}Downloading and installing startup unit space-ui.rc...${plain}"
        if ! cp -f space-ui.rc /etc/init.d/space-ui 2> /dev/null && ! fetch /etc/init.d/space-ui https://raw.githubusercontent.com/disp911/space-ui/${tag_version}/packaging/space-ui.rc; then
            _fail "ERROR: Failed to download startup unit space-ui.rc, please be sure that your server can access GitHub"
        fi
        chmod +x /etc/init.d/space-ui > /dev/null 2>&1
        chown root:root /etc/init.d/space-ui > /dev/null 2>&1
        rc-update add space-ui > /dev/null 2>&1
        rc-service space-ui start > /dev/null 2>&1
    else
        if [ -f "space-ui.service" ]; then
            echo -e "${green}Installing systemd unit...${plain}"
            cp -f space-ui.service ${space_ui_service}/ > /dev/null 2>&1
            if [[ $? -ne 0 ]]; then
                echo -e "${red}Failed to copy space-ui.service${plain}"
                exit 1
            fi
        else
            service_installed=false
            case "${release}" in
                ubuntu | debian | armbian)
                    if [ -f "space-ui.service.debian" ]; then
                        echo -e "${green}Installing debian-like systemd unit...${plain}"
                        cp -f space-ui.service.debian ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
                arch | manjaro | parch)
                    if [ -f "space-ui.service.arch" ]; then
                        echo -e "${green}Installing arch-like systemd unit...${plain}"
                        cp -f space-ui.service.arch ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
                *)
                    if [ -f "space-ui.service.rhel" ]; then
                        echo -e "${green}Installing rhel-like systemd unit...${plain}"
                        cp -f space-ui.service.rhel ${space_ui_service}/space-ui.service > /dev/null 2>&1
                        if [[ $? -eq 0 ]]; then
                            service_installed=true
                        fi
                    fi
                    ;;
            esac

            # If service file not found in tar.gz, download from GitHub
            if [ "$service_installed" = false ]; then
                echo -e "${yellow}Service files not found in tar.gz, downloading from GitHub...${plain}"
                case "${release}" in
                    ubuntu | debian | armbian)
                        fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${tag_version}/packaging/space-ui.service.debian
                        ;;
                    arch | manjaro | parch)
                        fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${tag_version}/packaging/space-ui.service.arch
                        ;;
                    *)
                        fetch ${space_ui_service}/space-ui.service https://raw.githubusercontent.com/disp911/space-ui/${tag_version}/packaging/space-ui.service.rhel
                        ;;
                esac

                if [[ $? -ne 0 ]]; then
                    echo -e "${red}Failed to install space-ui.service from GitHub${plain}"
                    exit 1
                fi
            fi
        fi
        chown root:root ${space_ui_service}/space-ui.service > /dev/null 2>&1
        chmod 644 ${space_ui_service}/space-ui.service > /dev/null 2>&1
        systemctl daemon-reload > /dev/null 2>&1
        systemctl enable space-ui > /dev/null 2>&1
        systemctl start space-ui > /dev/null 2>&1
    fi

    config_after_update

    echo -e "${green}space-ui ${tag_version}${plain} updating finished, it is running now..."
    /usr/bin/space-ui help
}

echo -e "${green}Running...${plain}"
install_base
update_space-ui $1
/usr/bin/space-ui site-status 2> /dev/null
