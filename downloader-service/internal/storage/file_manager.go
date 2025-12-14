package storage

import (
	"crypto/md5"
	"fmt"
	"io"
	"log"
	"os"
	"syscall"
)

// FileManager 文件管理器
type FileManager struct {
	basePath string
}

// NewFileManager 创建文件管理器
func NewFileManager(basePath string) *FileManager {
	return &FileManager{basePath: basePath}
}

// CalculateMD5 计算文件 MD5 哈希
func (m *FileManager) CalculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to calculate MD5: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// GetFileSize 获取文件大小
func (m *FileManager) GetFileSize(filePath string) (int64, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}

// DeleteFile 删除文件
func (m *FileManager) DeleteFile(filePath string) error {
	if err := os.Remove(filePath); err != nil {
		if os.IsNotExist(err) {
			log.Printf("[FileManager] File already deleted: %s", filePath)
			return nil
		}
		return fmt.Errorf("failed to delete file: %w", err)
	}
	log.Printf("[FileManager] Deleted file: %s", filePath)
	return nil
}

// DeleteDir 删除目录及其内容
func (m *FileManager) DeleteDir(dirPath string) error {
	if err := os.RemoveAll(dirPath); err != nil {
		return fmt.Errorf("failed to delete directory: %w", err)
	}
	log.Printf("[FileManager] Deleted directory: %s", dirPath)
	return nil
}

// FileExists 检查文件是否存在
func (m *FileManager) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return err == nil
}

// DiskUsage 磁盘使用情况
type DiskUsage struct {
	Total       uint64  // 总空间(字节)
	Available   uint64  // 可用空间(字节)
	Used        uint64  // 已用空间(字节)
	UsedPercent float64 // 使用百分比
}

// CheckDiskSpace 检查磁盘空间
func (m *FileManager) CheckDiskSpace(path string) (*DiskUsage, error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return nil, fmt.Errorf("failed to get disk stats: %w", err)
	}

	total := stat.Blocks * uint64(stat.Bsize)
	available := stat.Bavail * uint64(stat.Bsize)
	used := total - available
	usedPercent := float64(used) / float64(total) * 100

	return &DiskUsage{
		Total:       total,
		Available:   available,
		Used:        used,
		UsedPercent: usedPercent,
	}, nil
}

// IsDiskSpaceSufficient 检查磁盘空间是否充足
func (m *FileManager) IsDiskSpaceSufficient(path string, threshold float64) (bool, error) {
	usage, err := m.CheckDiskSpace(path)
	if err != nil {
		return false, err
	}

	if usage.UsedPercent > threshold {
		log.Printf("[FileManager] Disk usage warning: %.2f%% (threshold: %.2f%%)",
			usage.UsedPercent, threshold)
		return false, nil
	}

	return true, nil
}

// EnsureDir 确保目录存在
func (m *FileManager) EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}
