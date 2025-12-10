// 当前路径状态
let currentPath = '.';
// 保存原始文件列表数据，用于筛选
let originalFileList = [];

// 页面加载完成后执行
window.addEventListener('DOMContentLoaded', () => {
    // 从URL路径中读取文件路径（如果存在）
    let initialPath = extractPathFromURL();

    // 加载文件列表
    loadFileList(initialPath);

    // 监听文件上传表单提交
    const uploadForm = document.getElementById('upload-form');
    uploadForm.addEventListener('submit', handleFileUpload);

    // 监听筛选输入框变化
    const fileFilter = document.getElementById('file-filter');
    fileFilter.addEventListener('input', applyFileFilter);

    // 监听URL变化事件，确保前进/后退按钮或直接修改URL时能更新页面内容
    window.addEventListener('popstate', () => {
        try {
            const newPath = extractPathFromURL();
            // 只有当路径与当前路径不同时才重新加载
            if (newPath !== currentPath) {
                loadFileList(newPath);
            }
        } catch (e) {
            console.error('Failed to handle URL change:', e);
        }
    });
});

// 从URL中提取路径
function extractPathFromURL() {
    const path = window.location.pathname;
    if (path.startsWith('/files/')) {
        // 提取/files/后面的路径
        const extractedPath = path.substring(7);
        return extractedPath || '.';
    }
    return '.';
}

// 加载文件列表
async function loadFileList(path) {
    const fileListElement = document.getElementById('file-list');
    const pathNavElement = document.getElementById('path-nav');

    try {
        // 更新当前路径
        currentPath = path;

        // 更新URL路径以反映当前文件路径
        const newUrl = path === '.' ? '/' : `/files/${path}`;
        window.history.pushState({ path: path }, '', newUrl);

        // 构建请求URL
        const url = `/api/files?path=${encodeURIComponent(path)}`;
        const response = await fetch(url);

        if (!response.ok) {
            throw new Error(`Server response error: ${response.status}`);
        }

        const apiResponse = await response.json();

        // 更新路径导航
        updatePathNavigation(path);

        // 清空文件列表
        fileListElement.innerHTML = '';

        // 检查API响应状态
        if (!apiResponse || !apiResponse.success) {
            // API返回错误
            const errorMessage = apiResponse && apiResponse.error ?
                `Cannot access directory: ${apiResponse.error}` :
                'Cannot access directory: Server error';
            fileListElement.innerHTML = `<p class="error-message">${errorMessage}</p>`;
            return;
        }

        // 从响应中提取文件列表
        const files = apiResponse.data || [];

        // 保存原始文件列表
        originalFileList = files;

        // 检查文件列表是否为空
        if (files.length === 0) {
            fileListElement.innerHTML = '<p>No files in current directory</p>';
            return;
        }

        // 应用当前筛选条件
        const filteredFiles = applyFileFilterToFiles(originalFileList);

        // 创建文件列表项
        filteredFiles.forEach(file => {
            const fileItem = document.createElement('div');
            // 根据是否为目录设置不同的类名
            fileItem.className = file.isDir ? 'file-item directory' : 'file-item';

            // 格式化文件大小
            const formattedSize = file.isDir ? `${file.fileCount} items` : formatFileSize(file.size);

            // 格式化日期
            const formattedDate = new Date(file.modTime).toLocaleString();

            // 创建文件内容
            let fileContent;
            if (file.isDir) {
                // 目录项：点击进入子目录
                fileContent = `
                    <div class="file-info">
                        <div class="file-main">
                            <span class="file-icon">📁</span>
                            <a href="javascript:void(0)" class="file-name" data-name="${encodeURIComponent(file.name)}">
                                ${file.name}
                            </a>
                        </div>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">Modified: ${formattedDate}</div>
                `;
            } else {
                // 文件项：点击下载
                // 构建完整的文件路径（包含当前目录）
                const fullFilePath = path === '.' ? file.name : `${path}/${file.name}`;
                fileContent = `
                    <div class="file-info">
                        <div class="file-main">
                            <span class="file-icon">📄</span>
                            <a href="/files/${fullFilePath}" class="file-name" download>
                                ${file.name}
                            </a>
                        </div>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">Modified: ${formattedDate}</div>
                `;
            }

            fileItem.innerHTML = fileContent;
            fileListElement.appendChild(fileItem);
        });

        // 为所有目录项添加点击事件
        document.querySelectorAll('.file-item.directory .file-name').forEach(link => {
            link.addEventListener('click', (e) => {
                const dirName = decodeURIComponent(link.getAttribute('data-name'));
                // 构建新的路径
                const newPath = path === '.' ? dirName : `${path}/${dirName}`;
                loadFileList(newPath);
            });
        });

    } catch (error) {
        console.error('Failed to load file list:', error);
        fileListElement.innerHTML = `<p class="error">Failed to load file list: ${error.message}</p>`;
    }
}

// 应用文件筛选
function applyFileFilter() {
    const fileListElement = document.getElementById('file-list');

    // 清空文件列表
    fileListElement.innerHTML = '';

    // 应用筛选条件
    const filteredFiles = applyFileFilterToFiles(originalFileList);

    if (filteredFiles.length === 0) {
        fileListElement.innerHTML = '<p>No matching files</p>';
        return;
    }

    // 创建文件列表项
    filteredFiles.forEach(file => {
        const fileItem = document.createElement('div');
        // 根据是否为目录设置不同的类名
        fileItem.className = file.isDir ? 'file-item directory' : 'file-item';

        // 格式化文件大小
        const formattedSize = file.isDir ? `${file.fileCount} items` : formatFileSize(file.size);

        // 格式化日期
        const formattedDate = new Date(file.modTime).toLocaleString();

        // 创建文件内容
        let fileContent;
        if (file.isDir) {
            // 目录项：点击进入子目录
            fileContent = `
                    <div class="file-info">
                        <div class="file-main">
                            <span class="file-icon">📁</span>
                            <a href="javascript:void(0)" class="file-name" data-name="${encodeURIComponent(file.name)}">
                                ${file.name}
                            </a>
                        </div>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">Modified: ${formattedDate}</div>
                `;
        } else {
            // 文件项：点击下载
            // 构建完整的文件路径（包含当前目录）
            const fullFilePath = currentPath === '.' ? file.name : `${currentPath}/${file.name}`;
            fileContent = `
                    <div class="file-info">
                        <div class="file-main">
                            <span class="file-icon">📄</span>
                            <a href="/files/${fullFilePath}" class="file-name" download>
                                ${file.name}
                            </a>
                        </div>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">Modified: ${formattedDate}</div>
                `;
        }

        fileItem.innerHTML = fileContent;
        fileListElement.appendChild(fileItem);
    });

    // 为所有目录项添加点击事件
    document.querySelectorAll('.file-item.directory .file-name').forEach(link => {
        link.addEventListener('click', (e) => {
            const dirName = decodeURIComponent(link.getAttribute('data-name'));
            // 构建新的路径
            const newPath = currentPath === '.' ? dirName : `${currentPath}/${dirName}`;
            loadFileList(newPath);
        });
    });
}

// 根据筛选条件过滤文件
function applyFileFilterToFiles(files) {
    const filterInput = document.getElementById('file-filter');
    const filterText = filterInput.value.trim().toLowerCase();

    // 如果筛选条件为空，返回所有文件
    if (!filterText) {
        return files;
    }

    // 应用筛选条件
    return files.filter(file => {
        // 不区分大小写地比较文件名
        return file.name.toLowerCase().includes(filterText);
    });
}

// 更新路径导航
function updatePathNavigation(path) {
    const pathNavElement = document.getElementById('path-nav');

    // 构建路径部分数组
    const pathParts = path === '.' ? [] : path.split('/');

    // 创建导航HTML
    let navHtml = '<a href="javascript:void(0)" data-path=".">[Root]</a>';

    // 添加每个路径部分
    let currentSubPath = '.';
    for (let i = 0; i < pathParts.length; i++) {
        currentSubPath = i === 0 ? pathParts[i] : `${currentSubPath}/${pathParts[i]}`;
        navHtml += ` <span>/</span> 
                    <a href="javascript:void(0)" data-path="${encodeURIComponent(currentSubPath)}">
                        ${pathParts[i]}
                    </a>`;
    }

    // 设置导航HTML
    pathNavElement.innerHTML = navHtml;

    // 为导航链接添加点击事件
    document.querySelectorAll('#path-nav a').forEach(link => {
        link.addEventListener('click', (e) => {
            const targetPath = decodeURIComponent(link.getAttribute('data-path'));
            loadFileList(targetPath);
        });
    });
}

// 处理文件上传
async function handleFileUpload(event) {
    event.preventDefault();

    const fileInput = document.getElementById('file-input');
    const statusElement = document.getElementById('upload-status');

    if (fileInput.files.length === 0) {
        showStatus(statusElement, 'Please select files to upload', 'error');
        return;
    }

    // 为每个文件创建表单数据并上传
    for (let i = 0; i < fileInput.files.length; i++) {
        const file = fileInput.files[i];
        const formData = new FormData();
        formData.append('file', file);

        try {
            showStatus(statusElement, `Uploading ${file.name}...`, '');

            // 在FormData中添加当前路径信息
            formData.append('path', currentPath);

            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error(`Upload failed: ${response.status}`);
            }

            const result = await response.json();

            // 如果文件名被修改（因为冲突），显示新文件名
            const displayName = result.originalName !== result.savedName ?
                `${result.originalName} (renamed to: ${result.savedName})` :
                result.originalName;

            showStatus(statusElement, `File ${displayName} uploaded successfully`, 'success');

        } catch (error) {
            console.error('File upload failed:', error);
            showStatus(statusElement, `File ${file.name} upload failed: ${error.message}`, 'error');
            // Continue uploading other files without interruption
        }
    }

    // 上传完成后重新加载文件列表
    setTimeout(() => loadFileList(currentPath), 500);

    // 清空文件输入
    fileInput.value = '';
}

// 显示状态消息
function showStatus(element, message, type) {
    element.textContent = message;
    element.className = 'status';

    if (type) {
        element.classList.add(type);
    }
}

// 格式化文件大小
function formatFileSize(bytes) {
    if (bytes === 0) return '0 Bytes';

    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    // 确保索引在有效范围内
    const validIndex = Math.min(i, sizes.length - 1);

    return parseFloat((bytes / Math.pow(k, validIndex)).toFixed(2)) + ' ' + sizes[validIndex];
}