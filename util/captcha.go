package util

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"sync"
	"time"
)

// ========== 验证码存储 ==========

type captchaStore struct {
	mu       sync.RWMutex
	items    map[string]*captchaItem
	maxItems int
}

type captchaItem struct {
	Code      string
	ExpiresAt time.Time
}

var defaultCaptchaStore = &captchaStore{
	items:    make(map[string]*captchaItem),
	maxItems: 1000,
}

func init() {
	// 启动清理协程，每5分钟清理过期验证码
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			defaultCaptchaStore.cleanExpired()
		}
	}()
}

// Set 存储验证码
func (s *captchaStore) Set(id, code string, expire time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 防止内存无限增长，超过上限时清理一半
	if len(s.items) >= s.maxItems {
		s.cleanHalfLocked()
	}

	s.items[id] = &captchaItem{
		Code:      code,
		ExpiresAt: time.Now().Add(expire),
	}
}

// Get 获取并验证验证码（验证通过后延迟删除，防止并发请求误判）
func (s *captchaStore) Get(id, answer string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	item, exists := s.items[id]
	if !exists {
		return false
	}

	// 检查是否过期
	if time.Now().After(item.ExpiresAt) {
		delete(s.items, id)
		return false
	}

	// 验证码不匹配，直接返回 false（不删除，允许用户重试）
	if item.Code != answer {
		return false
	}

	// 验证通过：延迟 3 秒后删除，防止用户快速双击/并发请求导致第二个请求失败
	item.ExpiresAt = time.Now().Add(3 * time.Second)
	return true
}

// cleanExpired 清理过期验证码
func (s *captchaStore) cleanExpired() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for id, item := range s.items {
		if now.After(item.ExpiresAt) {
			delete(s.items, id)
		}
	}
}

// cleanHalfLocked 清理一半验证码（在锁内调用）
func (s *captchaStore) cleanHalfLocked() {
	count := 0
	target := len(s.items) / 2
	for id := range s.items {
		if count >= target {
			break
		}
		delete(s.items, id)
		count++
	}
}

// ========== 验证码生成器 ==========

const (
	captchaWidth  = 120
	captchaHeight = 40
	captchaLen    = 4
)

var (
	bgColor      = color.RGBA{245, 245, 245, 255}
	fontColors   = []color.RGBA{{0x22, 0x22, 0x22, 0xFF}, {0xCC, 0x33, 0x33, 0xFF}, {0x33, 0x66, 0xCC, 0xFF}, {0x33, 0x99, 0x33, 0xFF}}
	noiseColors  = []color.RGBA{{0xAA, 0xAA, 0xAA, 0xFF}, {0xCC, 0xCC, 0xCC, 0xFF}}
)

// GenerateCaptcha 生成验证码，返回验证码ID、验证码内容和base64图片
func GenerateCaptcha() (id, code, base64Img string) {
	id = GenerateUUID()
	code = randDigitCode(captchaLen)
	img := drawDigitCaptcha(code)
	base64Img = imageToBase64(img)
	defaultCaptchaStore.Set(id, code, 5*time.Minute)
	return
}

// VerifyCaptcha 验证验证码
func VerifyCaptcha(id, answer string) bool {
	if id == "" || answer == "" {
		return false
	}
	return defaultCaptchaStore.Get(id, answer)
}

// randDigitCode 生成随机数字验证码
func randDigitCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte('0' + rand.Intn(10))
	}
	return string(b)
}

// drawDigitCaptcha 绘制数字验证码图片
func drawDigitCaptcha(code string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))

	// 背景
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// 干扰线
	for i := 0; i < 6; i++ {
		drawNoiseLine(img)
	}

	// 干扰点
	for i := 0; i < 60; i++ {
		drawNoisePoint(img)
	}

	// 绘制每个数字
	charWidth := captchaWidth / len(code)
	for i, ch := range code {
		x := i*charWidth + charWidth/4 + rand.Intn(6)
		y := captchaHeight/2 + 5 + rand.Intn(4) - 2
		drawDigit(img, int(ch-'0'), x, y, i)
	}

	return img
}

// drawDigit 绘制单个数字（使用简单可靠的实心方块字体）
func drawDigit(img *image.RGBA, digit, x, y, idx int) {
	col := fontColors[idx%len(fontColors)]
	// 3x5 点阵数字
	bitmap := digitBitmap[digit]
	pixelSize := 3 + rand.Intn(2) // 3~4 像素
	offsetY := y - 10

	for row := 0; row < 5; row++ {
		for colIdx := 0; colIdx < 3; colIdx++ {
			if bitmap[row][colIdx] == 1 {
				px := x + colIdx*pixelSize
				py := offsetY + row*pixelSize
				fillRect(img, px, py, pixelSize, pixelSize, col)
			}
		}
	}
}

// fillRect 填充矩形
func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < captchaWidth && py >= 0 && py < captchaHeight {
				img.Set(px, py, c)
			}
		}
	}
}

// drawNoiseLine 绘制干扰线
func drawNoiseLine(img *image.RGBA) {
	c := noiseColors[rand.Intn(len(noiseColors))]
	x1 := rand.Intn(captchaWidth)
	y1 := rand.Intn(captchaHeight)
	x2 := rand.Intn(captchaWidth)
	y2 := rand.Intn(captchaHeight)

	steps := 30
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(float64(x1) + t*float64(x2-x1))
		y := int(float64(y1) + t*float64(y2-y1))
		if x >= 0 && x < captchaWidth && y >= 0 && y < captchaHeight {
			img.Set(x, y, c)
		}
	}
}

// drawNoisePoint 绘制干扰点
func drawNoisePoint(img *image.RGBA) {
	c := noiseColors[rand.Intn(len(noiseColors))]
	x := rand.Intn(captchaWidth)
	y := rand.Intn(captchaHeight)
	img.Set(x, y, c)
}

// imageToBase64 图片转base64
func imageToBase64(img image.Image) string {
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return ""
	}
	return "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())
}

// digitBitmap 0-9 的 3x5 点阵
var digitBitmap = [10][5][3]int{
	{{1, 1, 1}, {1, 0, 1}, {1, 0, 1}, {1, 0, 1}, {1, 1, 1}}, // 0
	{{0, 1, 0}, {1, 1, 0}, {0, 1, 0}, {0, 1, 0}, {1, 1, 1}}, // 1
	{{1, 1, 1}, {0, 0, 1}, {1, 1, 1}, {1, 0, 0}, {1, 1, 1}}, // 2
	{{1, 1, 1}, {0, 0, 1}, {1, 1, 1}, {0, 0, 1}, {1, 1, 1}}, // 3
	{{1, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 0, 1}, {0, 0, 1}}, // 4
	{{1, 1, 1}, {1, 0, 0}, {1, 1, 1}, {0, 0, 1}, {1, 1, 1}}, // 5
	{{1, 1, 1}, {1, 0, 0}, {1, 1, 1}, {1, 0, 1}, {1, 1, 1}}, // 6
	{{1, 1, 1}, {0, 0, 1}, {0, 0, 1}, {0, 0, 1}, {0, 0, 1}}, // 7
	{{1, 1, 1}, {1, 0, 1}, {1, 1, 1}, {1, 0, 1}, {1, 1, 1}}, // 8
	{{1, 1, 1}, {1, 0, 1}, {1, 1, 1}, {0, 0, 1}, {1, 1, 1}}, // 9
}
