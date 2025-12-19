#!/bin/bash

# 定义基础颜色
C_RESET="\033[0m"
C_RED="\033[31m"
C_GREEN="\033[32m"
C_YELLOW="\033[33m"
C_BLUE="\033[34m"
C_CYAN="\033[36m"

# 各色输出函数
log_red()    { printf "${C_RED}%s${C_RESET}\n" "$*"; }
log_green()  { printf "${C_GREEN}%s${C_RESET}\n" "$*"; }
log_yellow() { printf "${C_YELLOW}%s${C_RESET}\n" "$*"; }
log_blue()   { printf "${C_BLUE}%s${C_RESET}\n" "$*"; }
log_cyan()   { printf "${C_CYAN}%s${C_RESET}\n" "$*"; }
log()        { printf "%s\n" "$*"; }

# 读取确认函数
read_confirm()
{
    local prompt="$1"
    local ans
    read -p "${prompt} [y/N]: " ans
    case "$ans" in
        [Yy]) return 0 ;;  # 确认
        *)    return 1 ;;  # 取消
    esac
}

read_g_user_branch()
{
	local repo_name="${1}"
	local git_url="${2}"
	
	log "仓库:【${repo_name}】搜索【${BRANCH_NAME}】后的分支:"
	git ls-remote --heads "${git_url}" | grep "${BRANCH_NAME}"
	log ""
	while true; do

		read -p "请输入分支名称: " g_user_branch
		if [ -z "${g_user_branch}" ]; then
			log_cyan "未输入分支名称，请重新选择分支,以下是全部分支:"
			git ls-remote --heads "${git_url}"
			continue
		fi

		# 检查远程分支
		log_red "你输入的分支是: 【${g_user_branch}】"
		if git ls-remote --heads "${git_url}" "${g_user_branch}" | grep -q -w "refs/heads/$g_user_branch$"; then
			break
		else
			log "分支${g_user_branch}不存在,请重新选择分支,以下是全部分支:"
			git ls-remote --heads "${git_url}"
			continue
		fi
	done
	
}

modify_make_pack_ini()
{
	local repo_name="${1}"
	
	log "正在修改仓库${repo_name}的make_pack.ini..."
	cd "${ROOT_DIR}/${repo_name}"
	if [ -f "make_pack.ini" ]; then
		sed -i "s/^target_os_type=.*/target_os_type=${g_target_os_type}/" make_pack.ini
		local target_os_type=$(cat "${ROOT_DIR}/${repo_name}/make_pack.ini" | grep target_os_type)
		log_yellow "${repo_name}修改后的 make_pack.ini中的target_os_type为：${target_os_type}"
	else
		log_yellow "${repo_name}不存在make_pack.ini,不需要修改"
	fi
}

check_branch()
{
	local repo_name="${1}"
	local git_url="${2}"
	
	log ""
	log_blue "-----------------------------------------------------"
	log_blue "开始检出${repo_name}库..."
	read_g_user_branch "${repo_name}" "${git_url}"
	log_yellow "${repo_name}库你最终选择的分支是: 【${g_user_branch}】"

	# 目录检查
	if [ -d "${ROOT_DIR}/${repo_name}" ]; then
		if ! read_confirm "目录:${repo_name}已存在，是否要删除"; then
			log "用户选择：取消操作"
			exit 2
		fi
	fi

	rm -rf "${ROOT_DIR}/${repo_name}"
	sync
	
	cd "${ROOT_DIR}"
	if [ "${g_user_branch}" == "master" ]; then
		git clone "${git_url}"
	else
		git clone -b "${g_user_branch}" "${git_url}"
		echo $?
	fi
	
	
	if [ ! -d "${ROOT_DIR}/${repo_name}/.git" ]; then
		log_red "git clone ${git_url} fail."
		exit 3
	fi
	
	# 修改target_os_type
	modify_make_pack_ini "${repo_name}"
}

detect_os_arch()
{
    local os=""
    local version=""
    local arch=""

    # 1. 获取架构
    arch=$(uname -m)

    # 2. 获取操作系统信息
    if [ -f /etc/os-release ]; then
        # 现代系统通用路径
        . /etc/os-release
        os=$ID
        version=$VERSION_ID

    elif [ -f /etc/system-release ]; then
        sysrel=$(cat /etc/system-release)
        case "$sysrel" in
            *CentOS*6.*)  os="centos"; version="6" ;;
            *Rocky*Linux*) os="rocky"; version=$(echo "$sysrel" | grep -oE '[0-9]+(\.[0-9]+)?') ;;
            *Kylin*)      os="kylin"; version=$(echo "$sysrel" | grep -oE '[0-9]+(\.[0-9]+)?') ;;
            *UOS*|*uos*)  os="uos";   version=$(echo "$sysrel" | grep -oE '[0-9]+(\.[0-9]+)?') ;;
            *) os="unknown"; version="" ;;
        esac

    elif [ -f /etc/centos-release ]; then
        centrel=$(cat /etc/centos-release)
        if echo "$centrel" | grep -q "CentOS release 6"; then
            os="centos"; version="6"
        fi

    elif [ -f /etc/redhat-release ]; then
        redrel=$(cat /etc/redhat-release)
        case "$redrel" in
            *CentOS*6.*)  os="centos"; version="6" ;;
            *Rocky*Linux*) os="rocky"; version=$(echo "$redrel" | grep -oE '[0-9]+(\.[0-9]+)?') ;;
            *) os="unknown"; version="" ;;
        esac

    elif [ -f /etc/kylin-release ]; then
        os="kylin"
        version=$(grep -o '[0-9.]\+' /etc/kylin-release)

    elif [ -f /etc/uos-release ]; then
        os="uos"
        version=$(grep -o '[0-9.]\+' /etc/uos-release)

    else
        os="unknown"
        version=""
    fi
	
	log "获取到当前环境的os: ${os}, Version: ${version}, Arch: ${arch}"
	if [ "${os}" == "uos" ]; then
		g_target_os_type="uosv20a"
	elif [ "${os}" == "rocky" ]; then
		g_target_os_type="rockylinux8"
	fi
	
	log_yellow "最终确定的target_os_type为: ${g_target_os_type}"
	read -p "确定要继续么？"
}

make_cutil_code()
{
	local repo_name="${1}"
	
	if [ -z "${repo_name}" ]; then
		echo "make_cutil_code repo_name is empty."
		exit 3
	fi
	
	cd "${ROOT_DIR}/${repo_name}"
	
	cmake .
	cmake --build .
}

make_asm_07_common()
{
	local repo_name="${1}"
	
	if [ -z "${repo_name}" ]; then
		echo "make_asm_07_common repo_name is empty."
		exit 4
	fi
	
	cd "${ROOT_DIR}/${repo_name}"
	if [ -f "make_pack.ini" ]; then
		local ini_target_os_type=$(cat make_pack.ini  | grep target_os_type  | awk -F '=' '{print $2}')
		if [ "${ini_target_os_type}" != "${g_target_os_type}" ]; then
			echo "make_pack.ini target_os_type is ${ini_target_os_type}"
			exit 5
		fi
	fi
	
	cd src
	make clean_grpc_libs
    make  grpc_libs
   
    cd ../
    make clean
	make 
	
}

make_code_common()
{
	if [ -z "${repo_name}" ]; then
		echo "make_code_common repo_name ${repo_name} is empty."
		exit 4
	fi
	
	if [ ! -d "${ROOT_DIR}/${repo_name}" ]; then
		echo "make_code_common repo_name ${repo_name} directory is not exist."
		exit 5
	fi
	
	cd "${ROOT_DIR}/${repo_name}"
	if [ -f "make_pack.ini" ]; then
		local ini_target_os_type=$(cat make_pack.ini  | grep target_os_type  | awk -F '=' '{print $2}')
		if [ "${ini_target_os_type}" != "${g_target_os_type}" ]; then
			echo "make_pack.ini target_os_type is ${ini_target_os_type}"
			exit 5
		fi
	fi
	
	make clean
	make
	
}

# 操作系统环境
detect_os_arch
if [ -z "${g_target_os_type}" ]; then
	log_red "获取到的target_os_type为空，退出"
	exit 1
fi

# 根目录
ROOT_DIR=`pwd`
CUTIL_URL="git@git.infogo.tech:CBB/CUtil.git"
ASM_07_COMMON_URL="git@git.infogo.tech:ASM-Server/asm_07_common.git"

# 操作参数
MAKE_OPR="${1}"

# ASM仓库列表
ASM_REPO_ARR=(
    "git@git.infogo.tech:ASM-Server/agentless_check_svr.git"
    "git@git.infogo.tech:ASM-Server/agentless_collect_server.git"
    "git@git.infogo.tech:ASM-Server/agentless_collector.git"
    "git@git.infogo.tech:ASM-Server/agentless_snmp_mon.git"
    "git@git.infogo.tech:ASM-Server/asm_02_kernel.git"
    "git@git.infogo.tech:ASM-Server/asm_03_conf.git"
    "git@git.infogo.tech:ASM-Server/asm_03_firewall.git"
    "git@git.infogo.tech:ASM-Server/asm_03_init.git"
    "git@git.infogo.tech:ASM-Server/asm_03_regist.git"
    "git@git.infogo.tech:ASM-Server/asm_03_sysinit.git"
    "git@git.infogo.tech:ASM-Server/asm_04_tbserver.git"
    "git@git.infogo.tech:ASM-Server/asm_05_cli.git"
    "git@git.infogo.tech:ASM-Server/asm_06_ha.git"
    "git@git.infogo.tech:ASM-Server/asm_07_common.git"
    "git@git.infogo.tech:ASM-Server/asm_08_asmsrv.git"
    "git@git.infogo.tech:ASM-Server/asm_08_other.git"
    "git@git.infogo.tech:ASM-Server/asm_09_tool.git"
    "git@git.infogo.tech:ASM-Server/asm_10_netdiscover.git"
    "git@git.infogo.tech:ASM-Server/asm_12_dot1x.git"
    "git@git.infogo.tech:ASM-Server/asm_16_third_interface.git"
    "git@git.infogo.tech:ASM-Server/asm_17_ocean.git"
    "git@git.infogo.tech:ASM-Server/asm_20_ids.git"
    "git@git.infogo.tech:ASM-Server/asm_21_fingerprint.git"
    "git@git.infogo.tech:ASM-Server/asm_24_message.git"
    "git@git.infogo.tech:ASM-Server/asm_26_eventd.git"
    "git@git.infogo.tech:ASM-Server/asm_27_alarm_mng.git"
    "git@git.infogo.tech:ASM-Server/asm_30_repo.git"
    "git@git.infogo.tech:ASM-Server/asm_51_fwknop.git"
    "git@git.infogo.tech:ASM-Server/asm_52_gatewayvpn.git"
    "git@git.infogo.tech:ASM-Server/asm_53_terminalagent.git"
    "git@git.infogo.tech:ASM-Server/asm_54_httpagent.git"
    "git@git.infogo.tech:ASM-Server/asm_57_warnner.git"
    "git@git.infogo.tech:ASM-Server/asm_61_task.git"
    "git@git.infogo.tech:ASM-Server/asm_62_portal.git"
    "git@git.infogo.tech:ASM-Server/asm_63_LinuxExec.git"
    "git@git.infogo.tech:ASM-Server/asm_64_message_cat.git"
    "git@git.infogo.tech:ASM-Server/asm_65_NetSafe.git"
    "git@git.infogo.tech:ASM-Server/asm_66_sysgather.git"
    "git@git.infogo.tech:ASM-Server/asm_67_AswSvr.git"
    "git@git.infogo.tech:ASM-Server/asm_68_Lcd_Control.git"
    "git@git.infogo.tech:ASM-Server/asm_69_FlashSvr.git"
    "git@git.infogo.tech:ASM-Server/asm_70_PortTunnel.git"
    "git@git.infogo.tech:ASM-Server/asm_71_wdt_server.git"
    "git@git.infogo.tech:ASM-Server/asm_72_vpn_user.git"
    "git@git.infogo.tech:ASM-Server/asm_73_db_agent.git"
    "git@git.infogo.tech:ASM-Server/asm_74_RarPatch.git"
    "git@git.infogo.tech:ASM-Server/asm_75_UdpEchoServer.git"
    "git@git.infogo.tech:ASM-Server/asm_76_TerminalVisits.git"
    "git@git.infogo.tech:ASM-Server/asm_77_sysdebug_auth.git"
    "git@git.infogo.tech:ASM-Server/asm_78_WebsocketSvr.git"
    "git@git.infogo.tech:ASM-Server/asm_79_illegal_extranet.git"
    "git@git.infogo.tech:ASM-Server/asm_80_scencesvc.git"
    "git@git.infogo.tech:ASM-Server/asm_95_zabbix.git"
    "git@git.infogo.tech:ASM-Server/asm_events.git"
    "git@git.infogo.tech:ASM-Server/asm_make_software.git"
    "git@git.infogo.tech:ASM-Server/asm_server_reserved.git"
    "git@git.infogo.tech:ASM-Server/asm_syscheck.git"
    "git@git.infogo.tech:ASM-Server/asm_tassl.git"
    "git@git.infogo.tech:ASM-Server/asm_update.git"
    "git@git.infogo.tech:ASM-Server/asmproxy.git"
    "git@git.infogo.tech:ASM-Server/asmviripservice.git"
    "git@git.infogo.tech:ASM-Server/cascade_service.git"
    "git@git.infogo.tech:ASM-Server/docker-images.git"
    "git@git.infogo.tech:ASM-Server/isaproxylib.git"
    "git@git.infogo.tech:ASM-Server/proxy.git"
    "git@git.infogo.tech:ASM-Server/tas-service.git"
    "git@git.infogo.tech:ASM-Server/Temp_Liwn.git"
    "git@git.infogo.tech:ASM-Server/TestPR.git"
)

# 初始化环境
if [ "${MAKE_OPR}" == "init" ]; then
	
	
	if [ $# -ne 2 ]; then
		exit 1
	fi
	
	BRANCH_NAME="${2}"
	
	# 检出cutil
	check_branch "CUtil" "${CUTIL_URL}"

	# 检出asm_07_common库
	check_branch "asm_07_common" "${ASM_07_COMMON_URL}"

	# 编译cutil
	make_cutil_code "CUtil"

	# 编译 asm_07_common
	make_asm_07_common "asm_07_common"

elif [ "${MAKE_OPR}" == "repo" ]; then
	
	if [ $# -ne 3 ]; then
		exit 1
	fi

	# 需要导出的仓库
	REPO_NAME="${2}"
	BRANCH_NAME="${3}"
	
	# 取URL
	for url in "${ASM_REPO_ARR[@]}"; do
		repo_name=$(basename "$url" .git)
		if [ "${repo_name}" == "${REPO_NAME}" ]; then
			repo_url="${url}"
			break
		fi
	done
	
	log_yellow "仓库${REPO_NAME}的URL地址为:${repo_url}"
	check_branch "${REPO_NAME}" "${repo_url}"
	
elif [ "${MAKE_OPR}" == "branch" ]; then
	
	# 需要导出的仓库
	REPO_NAME="${2}"
	
	
	
	
	
	
	
	
	
	

fi
	
	
	

