// Package api 组装 HTTP 层：路由、处理器与鉴权中间件。
package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	"ShortLinkGo/server/utils"
)

// uploadRoot 上传文件根目录（通过 /uploads 静态服务）。
const uploadRoot = "./uploads"

// allowedImageExt 允许上传的图片扩展名。
var allowedImageExt = map[string]bool{
	".png": true, ".jpg": true, ".jpeg": true, ".gif": true, ".webp": true,
}

// allowedImageMime 允许的图片内容类型（http.DetectContentType 探测结果）。
var allowedImageMime = map[string]bool{
	"image/png": true, "image/jpeg": true, "image/gif": true, "image/webp": true,
}

// saveImage 保存表单字段 file 为图片，返回可访问的相对 URL。
func saveImage(c *gin.Context, sub string) (string, error) {
	file, err := c.FormFile("file")
	if err != nil {
		return "", errors.New("缺少上传文件")
	}
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !allowedImageExt[ext] {
		return "", errors.New("仅支持 png/jpg/jpeg/gif/webp 图片")
	}
	if file.Size > 2*1024*1024 {
		return "", errors.New("图片大小不能超过 2MB")
	}
	// 校验文件头魔数，拒绝伪装成图片的其他内容
	f, err := file.Open()
	if err != nil {
		return "", errors.New("无法读取上传文件")
	}
	defer f.Close()
	head := make([]byte, 512)
	n, _ := io.ReadFull(f, head)
	if n == 0 || !allowedImageMime[http.DetectContentType(head[:n])] {
		return "", errors.New("文件内容不是有效的图片")
	}
	dir := filepath.Join(uploadRoot, sub)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s%s", utils.RandomHex(8), ext)
	if err := c.SaveUploadedFile(file, filepath.Join(dir, name)); err != nil {
		return "", err
	}
	return "/uploads/" + sub + "/" + name, nil
}
