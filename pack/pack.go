package pack

import (
	"archive/zip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"go.uber.org/zap"
)

type ResourcePack struct {
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	Description  string    `json:"description"`
	PackFormat   int       `json:"pack_format"`
	Size         int64     `json:"size"`
	Hash         string    `json:"hash"`
	LastModified time.Time `json:"last_modified"`
	IsDirectory  bool      `json:"is_directory"`
}

func (rp *ResourcePack) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"name":          rp.Name,
		"description":   rp.Description,
		"pack_format":   rp.PackFormat,
		"size":          rp.Size,
		"hash":          rp.Hash,
		"last_modified": rp.LastModified.Unix(),
		"is_directory":  rp.IsDirectory,
		"download_url":  fmt.Sprintf("/download/%s", rp.Name),
		"hash_url":      fmt.Sprintf("/hash/%s", rp.Name),
	}
}

type PackInfo struct {
	Description string `json:"description"`
	PackFormat  int    `json:"pack_format"`
}

type PacksManager struct {
	config          *Config
	logger          *zap.Logger
	packsDirectory  string
	tempDir         string
	packs           map[string]*ResourcePack
	mu              sync.RWMutex
	fileWatcher     *fsnotify.Watcher
	fileMonitorStop chan struct{}
	lastScanTime    time.Time
	scanCooldown    time.Duration
}

type Config struct {
	Directory           string
	FileMonitor         bool
	FileMonitorInterval time.Duration
	ScanCooldown        time.Duration
}

func NewPacksManager(config *Config, logger *zap.Logger) (*PacksManager, error) {
	pm := &PacksManager{
		config:          config,
		logger:          logger,
		packsDirectory:  config.Directory,
		tempDir:         os.TempDir() + "/resourcepack_server",
		packs:           make(map[string]*ResourcePack),
		fileMonitorStop: make(chan struct{}),
		scanCooldown:    config.ScanCooldown,
	}

	if err := os.MkdirAll(pm.tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %w", err)
	}

	if err := os.MkdirAll(pm.packsDirectory, 0755); err != nil {
		return nil, fmt.Errorf("创建资源包目录失败: %w", err)
	}

	if err := pm.scanPacks(); err != nil {
		logger.Error("初始扫描资源包失败", zap.Error(err))
	}

	if config.FileMonitor {
		if err := pm.startFileMonitoring(); err != nil {
			logger.Error("启动文件监控失败", zap.Error(err))
		}
	}

	return pm, nil
}

func (pm *PacksManager) scanPacks() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	oldPacks := make(map[string]bool)
	for name := range pm.packs {
		oldPacks[name] = true
	}

	pm.packs = make(map[string]*ResourcePack)
	pm.logger.Info("开始扫描资源包目录", zap.String("directory", pm.packsDirectory))

	if pm.isResourcePackDirectory(pm.packsDirectory) {
		pack, err := pm.loadDirectoryPack(pm.packsDirectory)
		if err == nil && pack != nil {
			pm.packs[pack.Name] = pack
			pm.logger.Info("发现根目录资源包", zap.String("name", pack.Name))
		}
	}

	entries, err := os.ReadDir(pm.packsDirectory)
	if err != nil {
		return fmt.Errorf("读取目录失败: %w", err)
	}

	for _, entry := range entries {
		entryPath := filepath.Join(pm.packsDirectory, entry.Name())

		if entry.IsDir() {
			if pm.isResourcePackDirectory(entryPath) {
				pack, err := pm.loadDirectoryPack(entryPath)
				if err == nil && pack != nil {
					pm.packs[pack.Name] = pack
					pm.logger.Info("发现子目录资源包", zap.String("name", pack.Name))
				}
			}
		} else if strings.HasSuffix(entry.Name(), ".zip") {
			pack, err := pm.loadZipPack(entryPath)
			if err == nil && pack != nil {
				pm.packs[pack.Name] = pack
				pm.logger.Info("发现ZIP资源包", zap.String("name", pack.Name))
			}
		}
	}

	newPacks := make(map[string]bool)
	for name := range pm.packs {
		newPacks[name] = true
	}

	var added, removed []string
	for name := range newPacks {
		if !oldPacks[name] {
			added = append(added, name)
		}
	}
	for name := range oldPacks {
		if !newPacks[name] {
			removed = append(removed, name)
		}
	}

	if len(added) > 0 {
		pm.logger.Info("新增资源包", zap.Strings("names", added))
	}
	if len(removed) > 0 {
		pm.logger.Info("移除资源包", zap.Strings("names", removed))
	}

	pm.logger.Info("扫描完成", zap.Int("count", len(pm.packs)))
	pm.lastScanTime = time.Now()
	return nil
}

func (pm *PacksManager) isResourcePackDirectory(dirPath string) bool {
	packMcmetaPath := filepath.Join(dirPath, "pack.mcmeta")
	_, err := os.Stat(packMcmetaPath)
	return err == nil
}

func (pm *PacksManager) loadZipPack(packPath string) (*ResourcePack, error) {
	stat, err := os.Stat(packPath)
	if err != nil {
		return nil, err
	}

	name := strings.TrimSuffix(filepath.Base(packPath), ".zip")
	description := fmt.Sprintf("Resource Pack: %s", name)
	packFormat := 22

	if reader, err := zip.OpenReader(packPath); err == nil {
		defer reader.Close()
		for _, file := range reader.File {
			if file.Name == "pack.mcmeta" {
				if rc, err := file.Open(); err == nil {
					if content, err := io.ReadAll(rc); err == nil {
						if packInfo := pm.parsePackMcmeta(string(content)); packInfo != nil {
							description = packInfo.Description
							packFormat = packInfo.PackFormat
						}
					}
					rc.Close()
				}
				break
			}
		}
	}

	hash, err := pm.calculateFileHash(packPath)
	if err != nil {
		return nil, err
	}

	return &ResourcePack{
		Name:         name,
		Path:         packPath,
		Description:  description,
		PackFormat:   packFormat,
		Size:         stat.Size(),
		Hash:         hash,
		LastModified: stat.ModTime(),
		IsDirectory:  false,
	}, nil
}

func (pm *PacksManager) loadDirectoryPack(dirPath string) (*ResourcePack, error) {
	name := filepath.Base(dirPath)
	description := fmt.Sprintf("Resource Pack: %s", name)
	packFormat := 22

	packMcmetaPath := filepath.Join(dirPath, "pack.mcmeta")
	if content, err := os.ReadFile(packMcmetaPath); err == nil {
		if packInfo := pm.parsePackMcmeta(string(content)); packInfo != nil {
			description = packInfo.Description
			packFormat = packInfo.PackFormat
		}
	}

	size, err := pm.calculateDirectorySize(dirPath)
	if err != nil {
		return nil, err
	}

	hash, err := pm.calculateDirectoryHash(dirPath)
	if err != nil {
		return nil, err
	}

	stat, err := os.Stat(dirPath)
	if err != nil {
		return nil, err
	}

	return &ResourcePack{
		Name:         name,
		Path:         dirPath,
		Description:  description,
		PackFormat:   packFormat,
		Size:         size,
		Hash:         hash,
		LastModified: stat.ModTime(),
		IsDirectory:  true,
	}, nil
}

func (pm *PacksManager) parsePackMcmeta(content string) *PackInfo {
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		return nil
	}

	if pack, ok := data["pack"].(map[string]interface{}); ok {
		description := ""
		if desc, ok := pack["description"].(string); ok {
			description = desc
		}

		packFormat := 22
		if format, ok := pack["pack_format"].(float64); ok {
			packFormat = int(format)
		}

		return &PackInfo{
			Description: description,
			PackFormat:  packFormat,
		}
	}

	return nil
}

func (pm *PacksManager) calculateDirectorySize(dirPath string) (int64, error) {
	var totalSize int64
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	return totalSize, err
}

func (pm *PacksManager) calculateDirectoryHash(dirPath string) (string, error) {
	var fileInfos []string
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			relPath, _ := filepath.Rel(dirPath, path)
			fileInfos = append(fileInfos, fmt.Sprintf("%s:%d:%d", relPath, info.ModTime().Unix(), info.Size()))
		}
		return nil
	})
	if err != nil {
		return "", err
	}

	sort.Strings(fileInfos)
	content := strings.Join(fileInfos, "\n")
	hash := md5.Sum([]byte(content))
	return fmt.Sprintf("%x", hash), nil
}

func (pm *PacksManager) calculateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

func (pm *PacksManager) GetPack(name string) *ResourcePack {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.packs[name]
}

func (pm *PacksManager) GetAllPacks() []*ResourcePack {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	packs := make([]*ResourcePack, 0, len(pm.packs))
	for _, pack := range pm.packs {
		packs = append(packs, pack)
	}
	return packs
}

func (pm *PacksManager) GetPackHash(name string) string {
	pack := pm.GetPack(name)
	if pack != nil {
		return pack.Hash
	}
	return ""
}

func (pm *PacksManager) startFileMonitoring() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	pm.fileWatcher = watcher

	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				pm.handleFileEvent(event)
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				pm.logger.Error("文件监控错误", zap.Error(err))
			case <-pm.fileMonitorStop:
				return
			}
		}
	}()

	if err := watcher.Add(pm.packsDirectory); err != nil {
		return err
	}

	pm.logger.Info("文件监控已启动", zap.String("directory", pm.packsDirectory))
	return nil
}

func (pm *PacksManager) handleFileEvent(event fsnotify.Event) {
	if time.Since(pm.lastScanTime) < pm.scanCooldown {
		return
	}

	time.Sleep(500 * time.Millisecond)

	if err := pm.scanPacks(); err != nil {
		pm.logger.Error("文件变化后扫描失败", zap.Error(err))
	}
}

func (pm *PacksManager) StopFileMonitoring() {
	if pm.fileWatcher != nil {
		close(pm.fileMonitorStop)
		pm.fileWatcher.Close()
		pm.logger.Info("文件监控已停止")
	}
}

func (pm *PacksManager) CreateZipFromDirectory(dirPath, packName string) (string, error) {
	zipPath := filepath.Join(pm.tempDir, fmt.Sprintf("%s_%d.zip", packName, time.Now().Unix()))

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	err = filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		relPath, _ := filepath.Rel(dirPath, path)
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		defer file.Close()

		zipEntry, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		_, err = io.Copy(zipEntry, file)
		return err
	})

	if err != nil {
		return "", err
	}

	pm.logger.Info("已创建临时zip文件", zap.String("path", zipPath))
	return zipPath, nil
}

func (pm *PacksManager) GetPacksDirectory() string {
	return pm.packsDirectory
}

func (pm *PacksManager) RescanPacks() error {
	return pm.scanPacks()
}
