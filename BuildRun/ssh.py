#!/usr/bin/env python3
# -*- coding: utf-8 -*-

import os
import sys
import json
import subprocess
import configparser
import urllib.parse

import requests
import browser_cookie3
import ipaddress


# ================= 配置区 =================

DOMAIN = "infogo.tech"
INI_FILE = os.path.join(os.path.dirname(__file__), "config.ini")

XSHELL_PATHS = [
    r"C:\Program Files\NetSarang\Xshell 8\xshell.exe",
    r"C:\Program Files\NetSarang\Xshell 7\xshell.exe",
    r"C:\Program Files (x86)\NetSarang\Xshell 8\xshell.exe",
    r"C:\Program Files (x86)\NetSarang\Xshell 7\xshell.exe",
]


# ================= 工具函数 =================

def is_ip(addr: str) -> bool:
    try:
        ipaddress.ip_address(addr)
        return True
    except ValueError:
        return False

def find_xshell():
    for p in XSHELL_PATHS:
        if os.path.isfile(p):
            return p
    raise RuntimeError("未找到 Xshell.exe，请检查安装路径")


def load_ini():
    cfg = configparser.ConfigParser()
    if not os.path.exists(INI_FILE):
        return cfg

    for enc in ("gb2312", "utf-8"):
        try:
            cfg.read(INI_FILE, encoding=enc)
            return cfg
        except UnicodeDecodeError:
            continue

    raise RuntimeError("ssh_cache.ini 编码无法识别，请检查文件内容")


def save_ini(cfg):
    with open(INI_FILE, "w", encoding="gb2312") as f:
        cfg.write(f)


def find_host(cfg, name_or_ip):
    # 先读 [ssh] 下的 hosts 列表，再以 hosts 里的名字作为段名去找
    hosts = []
    if cfg.has_section("ssh"):
        raw = cfg.get("ssh", "hosts", fallback="")
        hosts = [x.strip() for x in raw.split(",") if x.strip()]
        
    # 2) 否则只在 hosts 列表对应的段里，按 host 字段匹配
    for sec in hosts:
        if not cfg.has_section(sec):
            continue
        if cfg.get(sec, "host", fallback="") == name_or_ip:
            return sec, dict(cfg[sec])
            
    return None, None


def update_cache(cfg, section, host, port, user, password):
    # 1) 从 [ssh].hosts 中找是否已存在相同 host
    hosts = []
    if cfg.has_section("ssh"):
        raw = cfg.get("ssh", "hosts", fallback="")
        hosts = [x.strip() for x in raw.split(",") if x.strip()]

    for sec in hosts:
        if not cfg.has_section(sec):
            continue
        if cfg.get(sec, "host", fallback="") == host:
            # 已存在：只更新密码
            cfg.set(sec, "pass", password)
            return

    # 2) 不存在：新增 section
    # 2.1 section 名已存在但 host 不同，属于冲突，拒绝新增
    if cfg.has_section(section):
        exist_host = cfg.get(section, "host", fallback="")
        if exist_host and exist_host != host:
            print(
                f"ERROR: section '{section}' 已存在(host={exist_host})，"
                f"拒绝绑定到新 host={host}",
                file=sys.stderr
            )
            return
    
    if not cfg.has_section(section):
        cfg.add_section(section)

    cfg.set(section, "host", host)
    cfg.set(section, "port", str(port))
    cfg.set(section, "user", user)
    cfg.set(section, "pass", password)
    cfg.set(section, "private_key", "")

    # 维护 [ssh].hosts
    if not cfg.has_section("ssh"):
        cfg.add_section("ssh")
        cfg.set("ssh", "hosts", section)
    else:
        if section not in hosts:
            hosts.append(section)
            cfg.set("ssh", "hosts", ",".join(hosts))

# ================= 业务逻辑 =================

def fetch_password_from_infogo(ip):
    try:
        cj = browser_cookie3.chrome(domain_name=DOMAIN)
        if not list(cj):
            raise RuntimeError("cookie 为空，请确认已登录 infogo.tech")

        url = f"https://{DOMAIN}/asm/{ip}"
        resp = requests.get(url, cookies=cj, timeout=10)

        if resp.status_code != 200:
            raise RuntimeError(f"HTTP {resp.status_code}")

        data = resp.json()
    except Exception as e:
        raise RuntimeError(f"请求 infogo 失败: {e}")

    if "password" in data:
        return data["password"]

    if isinstance(data.get("data"), dict) and "password" in data["data"]:
        return data["data"]["password"]

    raise RuntimeError("JSON 中未找到 password 字段")


def launch_xshell(ip, user, password, port=22):
    xshell = find_xshell()
    url = f"ssh://{user}:{password}@{ip}:{port}"
    subprocess.Popen([xshell, "-url", url], close_fds=True)


# ================= 主流程 =================

def main(target, serial=None):
    if not is_ip(target):
        password = fetch_password_from_infogo(target)
        print(password)
        return

    print("query cache password\n");
    cfg = load_ini()
    section, info = find_host(cfg, target)

    if info and info.get("pass"):
        print("query password: ", info["pass"])
        launch_xshell(
            info["host"],
            info.get("user", "root"),
            info["pass"],
            int(info.get("port", 22))
        )
        return
    
    # 缓存未命中：优先用 serial 查；serial 失败再回退用 IP 查
    password = None
    used_serial = False
    if serial is not None:
        try:
            print(f"query password by serial: {serial}\n")
            password = fetch_password_from_infogo(serial)
            used_serial = False  # 这里，如果想存，就改False，不想存，就改True
        except Exception as e:
            print(f"WARN: serial 查询失败，将回退用 IP 查询: {e}", file=sys.stderr)

    if password is None:
        print(f"query password by ip: {target}\n");
        password = fetch_password_from_infogo(target)
        used_serial = False
        
    print(f"query password: {password}")
    # 只有通过 IP 查询得到的密码才写缓存
    if not used_serial:
        update_cache(cfg, section or target, target, 22, "root", password)

    save_ini(cfg)
    launch_xshell(target, "root", password)


if __name__ == "__main__":
    if len(sys.argv) not in (2, 3):
        print(f"Usage:")
        print(f"  {sys.argv[0]} <ip>")
        print(f"  {sys.argv[0]} <ip> <serial>")
        print(f"  {sys.argv[0]} <serial>")
        sys.exit(1)

    try:
        if len(sys.argv) == 2:
            main(sys.argv[1])
        else:
            main(sys.argv[1], sys.argv[2])
    except Exception as e:
        print(f"ERROR: {e}", file=sys.stderr)
        sys.exit(2)
