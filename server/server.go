package server

import (
	"fmt"
	"net/http"
	"resourcepack-server/pack"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"resourcepack-server/config"
)

type Server struct {
	config       *config.Config
	packsManager *pack.PacksManager
	logger       *zap.Logger
	router       *gin.Engine
}

func NewServer(config *config.Config, packsManager *pack.PacksManager, logger *zap.Logger) *Server {
	if !config.Server.Debug {
		gin.SetMode(gin.ReleaseMode)
	}

	server := &Server{
		config:       config,
		packsManager: packsManager,
		logger:       logger,
		router:       gin.New(),
	}

	server.setupRoutes()
	return server
}

func (s *Server) setupRoutes() {
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(s.errorMiddleware())

	s.router.GET("/", s.indexHandler)
	s.router.GET("/api/packs", s.listPacksHandler)
	s.router.GET("/api/packs/:name", s.getPackHandler)
	s.router.GET("/download/:name", s.downloadPackHandler)
	s.router.GET("/hash/:name", s.hashHandler)
	s.router.GET("/api/rescan", s.rescanPacksHandler)
	s.router.GET("/debug", s.debugHandler)
}

func (s *Server) indexHandler(c *gin.Context) {
	resourcePacks := s.packsManager.GetAllPacks()

	htmlContent := fmt.Sprintf(`
<!DOCTYPE html>
<html lang="zh-CN">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Minecraft 资源包服务器</title>
    <style>
        body { font-family: 'Microsoft YaHei', sans-serif; margin: 0; padding: 20px; background: #f5f5f5; }
        .container { max-width: 800px; margin: 0 auto; background: white; padding: 30px; border-radius: 10px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #2c3e50; text-align: center; margin-bottom: 30px; }
        .pack-card { border: 1px solid #ddd; padding: 20px; margin-bottom: 20px; border-radius: 8px; background: #fafafa; }
        .pack-name { font-size: 1.3em; font-weight: bold; color: #2c3e50; margin-bottom: 10px; }
        .pack-desc { color: #7f8c8d; margin-bottom: 15px; }
        .pack-meta { font-size: 0.9em; color: #95a5a6; margin-bottom: 15px; }
        .download-btn, .copy-btn { 
            background: #27ae60; 
            color: white; 
            padding: 10px 20px; 
            text-decoration: none; 
            border-radius: 5px; 
            display: inline-block; 
            margin-right: 10px; 
            cursor: pointer; 
            border: none; 
            font-size: 14px; 
            font-family: inherit; 
            line-height: 1.4; 
            min-width: 120px; 
            text-align: center; 
        }
        .download-btn:hover { background: #229954; }
        .copy-btn { background: #3498db; }
        .copy-btn:hover { background: #2980b9; }
        .copy-btn:active { background: #1f5f8b; }
        .hash-info { background: #ecf0f1; padding: 10px; border-radius: 5px; font-family: monospace; font-size: 0.9em; }
        .no-resourcePacks { text-align: center; color: #7f8c8d; font-style: italic; }
        .copy-feedback { 
            position: fixed; 
            top: 20px; 
            right: 20px; 
            background: #27ae60; 
            color: white; 
            padding: 10px 20px; 
            border-radius: 5px; 
            display: none; 
            z-index: 1000; 
            animation: slideIn 0.3s ease-out; 
        }
        @keyframes slideIn {
            from { transform: translateX(100%%); opacity: 0; }
            to { transform: translateX(0); opacity: 1; }
        }
    </style>
    <script>
        function copyHash(hash) {
            navigator.clipboard.writeText(hash).then(function() {
                showCopyFeedback('Hash 已复制到剪贴板！');
            }).catch(function(err) {
                const textArea = document.createElement('textarea');
                textArea.value = hash;
                document.body.appendChild(textArea);
                textArea.select();
                try {
                    document.execCommand('copy');
                    showCopyFeedback('Hash 已复制到剪贴板！');
                } catch (err) {
                    showCopyFeedback('复制失败，请手动复制');
                }
                document.body.removeChild(textArea);
            });
        }
        
        function showCopyFeedback(message) {
            const feedback = document.getElementById('copy-feedback');
            feedback.textContent = message;
            feedback.style.display = 'block';
            
            setTimeout(function() {
                feedback.style.display = 'none';
            }, 2000);
        }
    </script>
</head>
<body>
    <div class="container">
        <h1>Minecraft 资源包服务器</h1>
        <p style="text-align: center; color: #7f8c8d; margin-bottom: 30px;">
        </p>
        
        <h2>可用资源包 (%d 个)</h2>
`, len(resourcePacks))

	if len(resourcePacks) == 0 {
		htmlContent += `<div class="no-resourcePacks">暂无可用资源包</div>`
	} else {
		for _, resourcePack := range resourcePacks {
			sizeMB := float64(resourcePack.Size) / 1024 / 1024
			htmlContent += fmt.Sprintf(`
        <div class="resourcePack-card">
            <div class="resourcePack-name">%s</div>
            <div class="resourcePack-desc">%s</div>
            <div class="resourcePack-meta">
                格式: %d | 大小: %.2f MB<br>
                类型: %s | 
                更新时间: %s
            </div>
            <div class="hash-info">
                <strong>Hash (MD5):</strong> %s
            </div>
            <a href="/download/%s" class="download-btn">下载资源包</a>
            <button onclick="copyHash('%s')" class="copy-btn">复制 Hash</button>
        </div>
`, resourcePack.Name, resourcePack.Description, resourcePack.PackFormat, sizeMB,
				func() string {
					if resourcePack.IsDirectory {
						return "目录"
					} else {
						return "ZIP文件"
					}
				}(),
				resourcePack.LastModified.Format("2006-01-02 15:04:05"), resourcePack.Hash, resourcePack.Name, resourcePack.Hash)
		}
	}

	htmlContent += `
    </div>
    <div id="copy-feedback" class="copy-feedback"></div>
    <div style="text-align: center; margin-top: 30px; padding: 20px; color: #7f8c8d; border-top: 1px solid #eee;">
        <p>&copy; 2025 Nipuru. All rights reserved.</p>
    </div>
</body>
</html>
`

	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, htmlContent)
}

func (s *Server) listPacksHandler(c *gin.Context) {
	resourcePacks := s.packsManager.GetAllPacks()
	packsData := make([]map[string]interface{}, 0, len(resourcePacks))

	for _, resourcePack := range resourcePacks {
		packsData = append(packsData, resourcePack.ToMap())
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    packsData,
		"count":   len(packsData),
	})
}

func (s *Server) getPackHandler(c *gin.Context) {
	name := c.Param("name")
	resourcePack := s.packsManager.GetPack(name)

	if resourcePack == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "资源包不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    resourcePack.ToMap(),
	})
}

func (s *Server) downloadPackHandler(c *gin.Context) {
	name := c.Param("name")
	resourcePack := s.packsManager.GetPack(name)

	if resourcePack == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "资源包不存在",
		})
		return
	}

	if resourcePack.IsDirectory {
		// 创建临时zip文件
		zipPath, err := s.packsManager.CreateZipFromDirectory(resourcePack.Path, resourcePack.Name)
		if err != nil {
			s.logger.Error("创建zip文件失败", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   "资源包文件生成失败",
			})
			return
		}

		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", resourcePack.Name))
		c.Header("Content-Type", "application/zip")
		c.File(zipPath)
	} else {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.zip\"", resourcePack.Name))
		c.Header("Content-Type", "application/zip")
		c.File(resourcePack.Path)
	}
}

func (s *Server) hashHandler(c *gin.Context) {
	name := c.Param("name")
	hash := s.packsManager.GetPackHash(name)

	if hash == "" {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "资源包不存在",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"name":      name,
			"hash":      hash,
			"hash_type": "MD5",
		},
	})
}

func (s *Server) rescanPacksHandler(c *gin.Context) {
	go func() {
		if err := s.packsManager.RescanPacks(); err != nil {
			s.logger.Error("重新扫描失败", zap.Error(err))
		}
	}()

	c.JSON(http.StatusOK, gin.H{
		"success":   true,
		"message":   "资源包重新扫描已启动",
		"timestamp": time.Now().Unix(),
	})
}

func (s *Server) debugHandler(c *gin.Context) {
	debugInfo := gin.H{
		"server":  "Resource Pack Server",
		"version": "1.0.0",
		"config": gin.H{
			"host":  s.config.Server.Host,
			"port":  s.config.Server.Port,
			"debug": s.config.Server.Debug,
		},
		"packs": gin.H{
			"directory": s.packsManager.GetPacksDirectory(),
			"count":     len(s.packsManager.GetAllPacks()),
		},
		"endpoints": gin.H{
			"list_packs": "/api/packs",
			"get_pack":   "/api/packs/{name}",
			"download":   "/download/{name}",
			"hash":       "/hash/{name}",
			"rescan":     "/api/rescan",
			"debug":      "/debug",
		},
		"timestamp": time.Now().Unix(),
	}

	c.JSON(http.StatusOK, debugInfo)
}

func (s *Server) errorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) > 0 {
			err := c.Errors.Last()
			s.logger.Error("请求处理错误", zap.Error(err.Err))

			c.JSON(http.StatusInternalServerError, gin.H{
				"success": false,
				"error":   err.Error(),
				"status":  500,
			})
		}
	}
}

func (s *Server) Run() error {
	addr := s.config.Server.Host + ":" + strconv.Itoa(s.config.Server.Port)
	s.logger.Info("启动HTTP服务器", zap.String("address", addr))
	return s.router.Run(addr)
}

func (s *Server) GetRouter() *gin.Engine {
	return s.router
}
