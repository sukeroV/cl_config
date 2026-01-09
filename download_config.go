package main

import (
	"archive/zip"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	repoURL := "https://github.com/sukeroV/cl_config.git"
	outputDir := "."

	result := downloadAndExtractConfig(repoURL, outputDir)
	if result != "" {
		fmt.Println(result)
		os.Exit(0)
	} else {
		fmt.Println("false")
		os.Exit(1)
	}
}

// downloadAndExtractConfig 下载并提取 YAML 文件
// 成功返回文件名，失败返回空字符串
func downloadAndExtractConfig(repoURL, outputDir string) string {
	// 解析仓库信息
	owner, repo, ok := parseRepoURL(repoURL)
	if !ok {
		return ""
	}

	// 确保输出目录存在
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return ""
	}

	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "gh_yaml_")
	if err != nil {
		return ""
	}
	defer os.RemoveAll(tempDir)

	// 尝试下载不同分支
	branches := []string{"main", "master"}
	var zipPath string
	var successBranch string

	for _, branch := range branches {
		zipURL := fmt.Sprintf("https://github.com/%s/%s/archive/refs/heads/%s.zip", owner, repo, branch)
		zipPath = filepath.Join(tempDir, repo+".zip")

		if err := downloadFile(zipURL, zipPath); err == nil {
			successBranch = branch
			break
		}
	}

	if successBranch == "" {
		return ""
	}

	// 解压 ZIP 文件
	if err := unzip(zipPath, tempDir); err != nil {
		return ""
	}

	// 找到解压后的根目录
	extractedDir, err := findExtractedDir(tempDir)
	if err != nil {
		return ""
	}

	// 查找 YAML 文件
	yamlFiles, err := findYAMLFiles(extractedDir)
	if err != nil {
		return ""
	}

	if len(yamlFiles) == 0 {
		return ""
	}

	// 复制 YAML 文件到输出目录
	for _, yamlPath := range yamlFiles {
		rel, err := filepath.Rel(extractedDir, yamlPath)
		if err != nil {
			continue
		}

		// 生成安全的文件名
		safeName := strings.ReplaceAll(rel, string(os.PathSeparator), "__")
		dest := filepath.Join(outputDir, safeName)

		if err := copyFile(yamlPath, dest); err != nil {
			continue
		}

		// 只返回第一个找到的 YAML 文件
		return filepath.Base(dest)
	}

	return ""
}

// 解析仓库 URL，返回 owner, repo, ok
func parseRepoURL(url string) (string, string, bool) {
	// 移除 .git 后缀
	url = strings.TrimSuffix(url, ".git")
	// 按 / 分割
	parts := strings.Split(url, "/")
	if len(parts) < 2 {
		return "", "", false
	}
	// 取最后两个部分作为 owner 和 repo
	owner := parts[len(parts)-2]
	repo := parts[len(parts)-1]
	return owner, repo, true
}

// 下载文件到指定路径
func downloadFile(url, path string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP 错误: %d", resp.StatusCode)
	}

	dst, err := os.Create(path)
	if err != nil {
		return err
	}
	defer dst.Close()

	_, err = io.Copy(dst, resp.Body)
	return err
}

// 解压 ZIP 文件到指定目录
func unzip(src, dest string) error {
	r, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		fpath := filepath.Join(dest, f.Name)

		// 创建目录
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(fpath, f.Mode()); err != nil {
				return err
			}
			continue
		}

		// 确保目录存在
		if err := os.MkdirAll(filepath.Dir(fpath), 0755); err != nil {
			return err
		}

		// 打开源文件
		inFile, err := f.Open()
		if err != nil {
			return err
		}

		// 创建目标文件
		outFile, err := os.Create(fpath)
		if err != nil {
			inFile.Close()
			return err
		}

		// 复制内容
		_, err = io.Copy(outFile, inFile)
		inFile.Close()
		outFile.Close()

		if err != nil {
			return err
		}

		// 设置权限
		if err := os.Chmod(fpath, f.Mode()); err != nil {
			return err
		}
	}

	return nil
}

// 找到解压后的根目录
func findExtractedDir(tempDir string) (string, error) {
	entries, err := os.ReadDir(tempDir)
	if err != nil {
		return "", err
	}

	for _, entry := range entries {
		if entry.IsDir() && strings.Contains(entry.Name(), "-") {
			return filepath.Join(tempDir, entry.Name()), nil
		}
	}

	return "", fmt.Errorf("未找到解压目录")
}

// 查找 YAML 文件
func findYAMLFiles(dir string) ([]string, error) {
	var yamlFiles []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			ext := strings.ToLower(filepath.Ext(path))
			if ext == ".yaml" || ext == ".yml" {
				yamlFiles = append(yamlFiles, path)
			}
		}

		return nil
	})

	return yamlFiles, err
}

// 复制文件
func copyFile(src, dest string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// 复制文件权限
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dest, srcInfo.Mode())
}
