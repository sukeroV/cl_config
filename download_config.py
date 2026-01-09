#!/usr/bin/env python3
# -*- coding: utf-8 -*-

"""
下载 GitHub 仓库并提取 YAML 配置文件（优化版）
"""

import os
import shutil
import tempfile
import zipfile
from urllib.parse import urlparse

# 尝试导入 requests 库，如果没有则使用 urllib
requests_available = False
try:
    import requests
    # 抑制 SSL 警告
    import urllib3
    urllib3.disable_warnings(urllib3.exceptions.InsecureRequestWarning)
    requests_available = True
except ImportError:
    import ssl
    import urllib.request


def get_repo_info(repo_url: str):
    """解析 owner / repo"""
    path = urlparse(repo_url).path.rstrip('/')
    owner, repo = path.split('/')[-2:]
    return owner, repo.replace('.git', '')


def download_zip(url, dest):
    """
    下载文件，优先使用 requests 库
    """
    try:
        if requests_available:
            # 使用 requests 库（更快）
            headers = {"User-Agent": "Mozilla/5.0"}
            # 禁用 SSL 验证以提高速度
            response = requests.get(url, headers=headers, stream=True, verify=False)
            if response.status_code != 200:
                return False
            # 分块下载，提高大文件下载速度
            with open(dest, 'wb') as f:
                for chunk in response.iter_content(chunk_size=8192):
                    if chunk:
                        f.write(chunk)
            return True
        else:
            # 回退到 urllib
            req = urllib.request.Request(
                url,
                headers={"User-Agent": "Mozilla/5.0"}
            )
            with urllib.request.urlopen(req, context=ssl._create_unverified_context()) as r:
                if r.status != 200:
                    return False
                with open(dest, 'wb') as f:
                    f.write(r.read())
            return True
    except Exception:
        return False


def download_and_extract_config(repo_url, output_dir):
    """
    下载并提取 YAML 文件
    成功返回文件名，失败返回空字符串
    """
    os.makedirs(output_dir, exist_ok=True)

    try:
        owner, repo = get_repo_info(repo_url)
    except:
        return ""

    branches = ["main", "master"]

    with tempfile.TemporaryDirectory(prefix="gh_yaml_") as temp_dir:
        zip_path = os.path.join(temp_dir, f"{repo}.zip")

        success = False
        for branch in branches:
            zip_url = f"https://github.com/{owner}/{repo}/archive/refs/heads/{branch}.zip"
            if download_zip(zip_url, zip_path):
                success = True
                break
        if not success:
            return ""

        try:
            with zipfile.ZipFile(zip_path) as z:
                z.extractall(temp_dir)
        except zipfile.BadZipFile:
            return ""

        try:
            extracted_root = next(
                d for d in os.listdir(temp_dir)
                if os.path.isdir(os.path.join(temp_dir, d))
            )
        except:
            return ""

        extracted_dir = os.path.join(temp_dir, extracted_root)

        yaml_files = []
        for root, _, files in os.walk(extracted_dir):
            for f in files:
                if f.endswith(('.yml', '.yaml')):
                    yaml_files.append(os.path.join(root, f))

        if not yaml_files:
            return ""

        for path in yaml_files:
            rel = os.path.relpath(path, extracted_dir)
            safe_name = rel.replace(os.sep, "__")
            dest = os.path.join(output_dir, safe_name)
            try:
                shutil.copy2(path, dest)
                return os.path.basename(dest)
            except:
                continue

        return ""


if __name__ == "__main__":
    REPO_URL = "https://github.com/sukeroV/cl_config.git"
    OUTPUT_DIR = os.path.dirname(os.path.abspath(__file__))

    result = download_and_extract_config(REPO_URL, OUTPUT_DIR)
    if result:
        print(result)
    else:
        print("false")
