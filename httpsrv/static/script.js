// 当前路径状态
let currentPath = '.';
// 保存原始文件列表数据，用于筛选
let originalFileList = [];

// 页面加载完成后执行
window.addEventListener('DOMContentLoaded', () => {
    // 从URL hash中读取文件路径（如果存在）
    let initialPath = currentPath;
    if (window.location.hash) {
        try {
            const hashPath = decodeURIComponent(window.location.hash.substring(1));
            if (hashPath) {
                initialPath = hashPath;
            }
        } catch (e) {
            console.error('解析URL hash失败:', e);
        }
    }

    // 加载文件列表
    loadFileList(initialPath);

    // 监听文件上传表单提交
    const uploadForm = document.getElementById('upload-form');
    uploadForm.addEventListener('submit', handleFileUpload);

    // 监听筛选输入框变化
    const fileFilter = document.getElementById('file-filter');
    fileFilter.addEventListener('input', applyFileFilter);

    // 监听URL hash变化事件，确保前进/后退按钮或直接修改URL时能更新页面内容
    window.addEventListener('hashchange', () => {
        try {
            let newPath = '.';
            if (window.location.hash) {
                const hashPath = decodeURIComponent(window.location.hash.substring(1));
                if (hashPath) {
                    newPath = hashPath;
                }
            }
            // 只有当hash路径与当前路径不同时才重新加载
            if (newPath !== currentPath) {
                loadFileList(newPath);
            }
        } catch (e) {
            console.error('处理URL hash变化失败:', e);
        }
    });
});

// 加载文件列表
async function loadFileList(path) {
    const fileListElement = document.getElementById('file-list');
    const pathNavElement = document.getElementById('path-nav');

    try {
        // 更新当前路径
        currentPath = path;

        // 更新URL的hash部分以反映当前文件路径
        window.location.hash = path === '.' ? '' : `#${encodeURIComponent(path)}`;

        // 构建请求URL
        const url = `/api/files?path=${encodeURIComponent(path)}`;
        const response = await fetch(url);

        if (!response.ok) {
            throw new Error(`服务器响应错误: ${response.status}`);
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
                `无法访问目录: ${apiResponse.error}` :
                '无法访问目录: 服务器错误';
            fileListElement.innerHTML = `<p class="error-message">${errorMessage}</p>`;
            return;
        }

        // 从响应中提取文件列表
        const files = apiResponse.data || [];

        // 保存原始文件列表
        originalFileList = files;

        // 检查文件列表是否为空
        if (files.length === 0) {
            fileListElement.innerHTML = '<p>当前目录没有文件</p>';
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
                        <a href="javascript:void(0)" class="file-name" data-name="${encodeURIComponent(file.name)}">
                            <span class="file-icon">📁</span>${file.name}
                        </a>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">修改时间: ${formattedDate}</div>
                `;
            } else {
                // 文件项：点击下载
                // 构建完整的文件路径（包含当前目录）
                const fullFilePath = path === '.' ? file.name : `${path}/${file.name}`;
                fileContent = `
                    <div class="file-info">
                        <a href="/files/${encodeURIComponent(fullFilePath)}" class="file-name" download>
                            <span class="file-icon">📄</span>${file.name}
                        </a>
                        <span class="file-size">${formattedSize}</span>
                    </div>
                    <div class="file-date">修改时间: ${formattedDate}</div>
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
        console.error('加载文件列表失败:', error);
        fileListElement.innerHTML = `<p class="error">加载文件列表失败: ${error.message}</p>`;
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
        fileListElement.innerHTML = '<p>没有匹配的文件</p>';
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
                    <a href="javascript:void(0)" class="file-name" data-name="${encodeURIComponent(file.name)}">
                        <span class="file-icon">📁</span>${file.name}
                    </a>
                    <span class="file-size">${formattedSize}</span>
                </div>
                <div class="file-date">修改时间: ${formattedDate}</div>
            `;
        } else {
            // 文件项：点击下载
            // 构建完整的文件路径（包含当前目录）
            const fullFilePath = currentPath === '.' ? file.name : `${currentPath}/${file.name}`;
            fileContent = `
                <div class="file-info">
                    <a href="/files/${encodeURIComponent(fullFilePath)}" class="file-name" download>
                        <span class="file-icon">📄</span>${file.name}
                    </a>
                    <span class="file-size">${formattedSize}</span>
                </div>
                <div class="file-date">修改时间: ${formattedDate}</div>
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
    let navHtml = '<a href="javascript:void(0)" data-path=".">根目录</a>';

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
        showStatus(statusElement, '请选择要上传的文件', 'error');
        return;
    }

    // 为每个文件创建表单数据并上传
    for (let i = 0; i < fileInput.files.length; i++) {
        const file = fileInput.files[i];
        const formData = new FormData();
        formData.append('file', file);

        try {
            showStatus(statusElement, `正在上传 ${file.name}...`, '');

            // 在FormData中添加当前路径信息
            formData.append('path', currentPath);

            const response = await fetch('/api/upload', {
                method: 'POST',
                body: formData
            });

            if (!response.ok) {
                throw new Error(`上传失败: ${response.status}`);
            }

            const result = await response.json();

            // 如果文件名被修改（因为冲突），显示新文件名
            const displayName = result.originalName !== result.savedName ?
                `${result.originalName} (已重命名为: ${result.savedName})` :
                result.originalName;

            showStatus(statusElement, `文件 ${displayName} 上传成功`, 'success');

        } catch (error) {
            console.error('文件上传失败:', error);
            showStatus(statusElement, `文件 ${file.name} 上传失败: ${error.message}`, 'error');
            // 继续上传其他文件，不中断
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