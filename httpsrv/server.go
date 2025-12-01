package httpsrv

import (
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 嵌入静态文件
//
//go:embed static/*
var staticFiles embed.FS

// 文件信息结构体
type FileInfo struct {
	Name      string    `json:"name"`
	Size      int64     `json:"size"`
	ModTime   time.Time `json:"modTime"`
	IsDir     bool      `json:"isDir"`
	FileCount int       `json:"fileCount"`
}

// 上传响应结构体
type UploadResponse struct {
	OriginalName string `json:"originalName"`
	SavedName    string `json:"savedName"`
	Size         int64  `json:"size"`
	Success      bool   `json:"success"`
}

// StartServer 启动HTTP文件服务器
func StartServer(listenAddress string) error {
	// 设置路由
	http.HandleFunc("/", serveStaticFile)
	http.HandleFunc("/api/files", handleGetFileList)
	http.HandleFunc("/api/upload", handleFileUpload)
	http.HandleFunc("/files/", serveFileDownload)

	// 获取当前工作目录
	workDir, err := os.Getwd()
	if err != nil {
		log.Printf("警告: 无法获取当前工作目录: %v", err)
		workDir = "未知目录"
	}

	// 启动服务器
	log.Printf("[INFO] 启动HTTP文件服务器")
	log.Printf("[INFO] 监听地址: %s", listenAddress)
	log.Printf("[INFO] 文件服务目录: %s", workDir)
	return http.ListenAndServe(listenAddress, nil)
}

// serveStaticFile 提供静态文件服务
func serveStaticFile(w http.ResponseWriter, r *http.Request) {
	// 如果路径是根路径，重定向到index.html
	path := r.URL.Path
	if path == "/" {
		path = "/static/index.html"
	} else if !strings.HasPrefix(path, "/static/") {
		// 对于非/static路径，尝试在static目录下查找
		path = "/static" + path
	}

	// 从嵌入的文件系统中读取文件
	content, err := staticFiles.ReadFile(strings.TrimPrefix(path, "/"))
	if err != nil {
		// 文件不存在，返回404
		http.NotFound(w, r)
		return
	}

	// 设置适当的Content-Type
	ext := filepath.Ext(path)
	switch ext {
	case ".html":
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case ".css":
		w.Header().Set("Content-Type", "text/css")
	case ".js":
		w.Header().Set("Content-Type", "application/javascript")
	}

	// 写入响应
	w.Write(content)
}

// APIResponse 定义统一的API响应格式
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

// handleGetFileList 处理获取文件列表请求
func handleGetFileList(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	log.Printf("[INFO] %s 请求文件列表", clientIP)

	if r.Method != http.MethodGet {
		log.Printf("[ERROR] %s 方法不允许，期望GET，实际为 %s", clientIP, r.Method)
		http.Error(w, "只支持GET请求", http.StatusMethodNotAllowed)
		return
	}

	// 获取请求的子目录参数
	subdir := r.URL.Query().Get("path")
	if subdir == "" {
		subdir = "."
	}

	// 获取指定目录的文件列表
	files, err := getDirectoryFiles(subdir)

	// 设置响应头
	w.Header().Set("Content-Type", "application/json")

	// 准备响应
	response := APIResponse{}

	if err != nil {
		log.Printf("[ERROR] %s 获取文件列表失败: %v", clientIP, err)
		// 返回错误信息，而不是使用http.Error
		response.Success = false
		response.Error = err.Error()
		response.Data = []FileInfo{} // 确保data字段始终是数组
	} else {
		// 成功响应
		response.Success = true
		// 确保data字段始终是数组，即使为nil
		if files == nil {
			response.Data = []FileInfo{}
		} else {
			response.Data = files
			log.Printf("[INFO] %s 成功获取文件列表，共 %d 个文件", clientIP, len(files))
		}
	}

	// 返回JSON响应
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] %s 编码文件列表响应失败: %v", clientIP, err)
		http.Error(w, "处理响应失败", http.StatusInternalServerError)
		return
	}
}

// getDirectoryFiles 获取指定目录的文件列表
func getDirectoryFiles(subdir string) ([]FileInfo, error) {
	// 获取当前工作目录
	baseDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取当前目录失败: %w", err)
	}

	// 构建完整路径
	targetDir := filepath.Join(baseDir, subdir)

	// 安全检查：确保不会访问到工作目录之外
	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		return nil, fmt.Errorf("获取绝对路径失败: %w", err)
	}

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		return nil, fmt.Errorf("获取基础目录绝对路径失败: %w", err)
	}

	// 确保目标目录在基础目录之内
	if !strings.HasPrefix(targetDir, baseDir) {
		return nil, fmt.Errorf("访问被拒绝：不能访问工作目录之外的路径")
	}

	// 读取目录内容
	dirEntries, err := os.ReadDir(targetDir)
	if err != nil {
		return nil, fmt.Errorf("读取目录失败: %w", err)
	}

	// 转换为文件信息列表
	var files []FileInfo
	for _, entry := range dirEntries {
		info, err := entry.Info()
		if err != nil {
			continue // 跳过无法获取信息的文件
		}

		// 不再跳过隐藏文件和目录
		fileInfo := FileInfo{
			Name:      info.Name(),
			Size:      info.Size(),
			ModTime:   info.ModTime(),
			IsDir:     info.IsDir(),
			FileCount: 0,
		}

		// 如果是目录，计算其中的条目数量（文件和文件夹）
		if info.IsDir() {
			dirPath := filepath.Join(targetDir, info.Name())
			fileInfo.FileCount = countFilesInDirectory(dirPath, baseDir)
		}

		files = append(files, fileInfo)
	}

	// 按文件夹优先排序，然后按名称排序
	sortFileList(files)

	return files, nil
}

// countFilesInDirectory 计算目录中的条目数量（文件和文件夹）
func countFilesInDirectory(dirPath string, baseDir string) int {
	// 安全检查：确保不会访问到工作目录之外
	dirPath, err := filepath.Abs(dirPath)
	if err != nil || !strings.HasPrefix(dirPath, baseDir) {
		return 0
	}

	// 读取当前目录内容
	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		return 0
	}

	// 返回所有条目的数量（包括文件和文件夹，不跳过隐藏文件）
	return len(dirEntries)
}

// sortFileList 按文件夹优先，然后按名称排序
func sortFileList(files []FileInfo) {
	sort.Slice(files, func(i, j int) bool {
		// 文件夹优先
		if files[i].IsDir != files[j].IsDir {
			return files[i].IsDir // true 在 false 前面
		}
		// 相同类型按名称排序（忽略大小写）
		return strings.ToLower(files[i].Name) < strings.ToLower(files[j].Name)
	})
}

// handleFileUpload 处理文件上传
func handleFileUpload(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr
	log.Printf("[INFO] %s 请求上传文件", clientIP)

	if r.Method != http.MethodPost {
		log.Printf("[ERROR] %s 方法不允许，期望POST，实际为 %s", clientIP, r.Method)
		http.Error(w, "只支持POST请求", http.StatusMethodNotAllowed)
		return
	}

	// 解析多部分表单
	err := r.ParseMultipartForm(10 << 20) // 10MB
	if err != nil {
		log.Printf("[ERROR] %s 解析表单失败: %v", clientIP, err)
		http.Error(w, "解析表单失败", http.StatusBadRequest)
		return
	}

	// 获取上传的文件
	file, header, err := r.FormFile("file")
	if err != nil {
		log.Printf("[ERROR] %s 获取上传文件失败: %v", clientIP, err)
		http.Error(w, "获取上传文件失败", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// 获取目标路径（从表单中读取）
	targetPath := r.FormValue("path")
	if targetPath == "" {
		targetPath = "." // 默认使用当前目录
	}

	// 获取当前工作目录
	baseDir, err := os.Getwd()
	if err != nil {
		log.Printf("[ERROR] %s 获取当前目录失败: %v", clientIP, err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	// 构建完整路径
	targetDir := filepath.Join(baseDir, targetPath)

	// 安全检查：确保不会访问到工作目录之外
	targetDir, err = filepath.Abs(targetDir)
	if err != nil {
		log.Printf("[ERROR] %s 获取绝对路径失败: %v", clientIP, err)
		http.Error(w, "无效的路径", http.StatusBadRequest)
		return
	}

	baseDir, err = filepath.Abs(baseDir)
	if err != nil {
		log.Printf("[ERROR] %s 获取基础目录绝对路径失败: %v", clientIP, err)
		http.Error(w, "服务器错误", http.StatusInternalServerError)
		return
	}

	// 确保目标目录在基础目录之内
	if !strings.HasPrefix(targetDir, baseDir) {
		log.Printf("[ERROR] %s 尝试访问工作目录之外的路径: %s", clientIP, targetPath)
		http.Error(w, "访问被拒绝：不能访问工作目录之外的路径", http.StatusForbidden)
		return
	}

	// 确保目标目录存在
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		log.Printf("[ERROR] %s 创建目录失败: %v", clientIP, err)
		http.Error(w, "创建目录失败", http.StatusInternalServerError)
		return
	}

	// 生成安全的文件名并处理重名
	savedFilename := getSafeFilename(header.Filename, targetDir)

	// 创建目标文件
	dst, err := os.Create(filepath.Join(targetDir, savedFilename))
	if err != nil {
		log.Printf("[ERROR] %s 创建文件失败: %v", clientIP, err)
		http.Error(w, "创建文件失败", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// 复制文件内容
	size, err := io.Copy(dst, file)
	if err != nil {
		log.Printf("[ERROR] %s 保存文件失败: %v", clientIP, err)
		http.Error(w, "保存文件失败", http.StatusInternalServerError)
		return
	}

	// 记录上传日志
	log.Printf("[INFO] %s 文件上传成功: 原始名称=%s, 保存名称=%s, 大小=%d bytes",
		clientIP, header.Filename, savedFilename, size)

	// 返回成功响应
	response := UploadResponse{
		OriginalName: header.Filename,
		SavedName:    savedFilename,
		Size:         size,
		Success:      true,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		log.Printf("[ERROR] %s 编码上传响应失败: %v", clientIP, err)
	}
}

// getSafeFilename 生成安全的文件名并处理重名
func getSafeFilename(originalFilename string, targetDir string) string {
	// 移除路径部分，只保留文件名
	filename := filepath.Base(originalFilename)

	// 移除可能的危险字符
	filename = strings.ReplaceAll(filename, "/", "_")
	filename = strings.ReplaceAll(filename, "\\", "_")

	// 构建完整路径
	fullPath := filepath.Join(targetDir, filename)

	// 检查文件是否存在，如果存在则添加计数器后缀
	ext := filepath.Ext(filename)
	nameWithoutExt := strings.TrimSuffix(filename, ext)

	counter := 1
	for {
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			return filename
		}

		// 生成新的文件名
		newFilename := fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext)
		fullPath = filepath.Join(targetDir, newFilename)
		filename = newFilename
		counter++
	}
}

// serveFileDownload 提供文件下载服务
func serveFileDownload(w http.ResponseWriter, r *http.Request) {
	clientIP := r.RemoteAddr

	// 提取文件路径（去掉/files/前缀）
	filePath := strings.TrimPrefix(r.URL.Path, "/files/")
	if filePath == "" {
		log.Printf("[WARNING] %s 请求无效的文件路径", clientIP)
		http.NotFound(w, r)
		return
	}

	// 安全检查和路径处理
	filePath = filepath.Clean(filePath)
	if filePath == ".." || strings.HasPrefix(filePath, "../") {
		log.Printf("[WARNING] %s 尝试访问非法路径", clientIP)
		http.Error(w, "访问被拒绝", http.StatusForbidden)
		return
	}

	log.Printf("[INFO] %s 请求下载文件: %s", clientIP, filePath)

	// 打开文件
	file, err := os.Open(filePath)
	if err != nil {
		log.Printf("[ERROR] %s 打开文件失败: %s - %v", clientIP, filePath, err)
		http.NotFound(w, r)
		return
	}
	defer file.Close()

	// 获取文件信息
	fileInfo, err := file.Stat()
	if err != nil {
		log.Printf("[ERROR] %s 获取文件信息失败: %s - %v", clientIP, filePath, err)
		http.Error(w, "获取文件信息失败", http.StatusInternalServerError)
		return
	}

	// 检查是否为目录
	if fileInfo.IsDir() {
		log.Printf("[ERROR] %s 尝试下载目录: %s", clientIP, filePath)
		http.Error(w, "不能下载目录", http.StatusBadRequest)
		return
	}

	// 设置响应头 - 使用文件名部分作为下载名称
	displayName := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", displayName))
	w.Header().Set("Content-Length", fmt.Sprintf("%d", fileInfo.Size()))

	// 尝试检测内容类型
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		log.Printf("[WARNING] %s 读取文件头部失败: %v", clientIP, err)
	} else {
		// 重置文件指针到开始位置
		_, seekErr := file.Seek(0, 0)
		if seekErr != nil {
			log.Printf("[ERROR] %s 重置文件指针失败: %v", clientIP, seekErr)
		}
		contentType := http.DetectContentType(buffer)
		w.Header().Set("Content-Type", contentType)
	}

	// 发送文件
	_, err = io.Copy(w, file)
	if err != nil {
		log.Printf("[ERROR] %s 文件传输失败: %s - %v", clientIP, filePath, err)
	} else {
		// 记录成功的下载日志
		log.Printf("[INFO] %s 文件下载完成: %s, 大小: %d bytes",
			clientIP, filePath, fileInfo.Size())
	}
}

// 辅助函数：保存上传的文件
func saveUploadedFile(file *multipart.FileHeader, dst string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, src)
	return err
}
