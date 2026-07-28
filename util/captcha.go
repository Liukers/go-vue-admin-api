package util

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"crypto/rand"
	"math/big"
	"sort"
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
	CreatedAt time.Time
	ExpiresAt time.Time
	FailCount int
}

// maxCaptchaFailCount 单个验证码允许的最大错误尝试次数（达到即作废，防止无限猜测）
const maxCaptchaFailCount = 5

var defaultCaptchaStore = &captchaStore{
	items:    make(map[string]*captchaItem),
	maxItems: 1000,
}

func init() {
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

	// 防止内存无限增长：超过上限时按创建时间淘汰最旧的10%，
	// 不再随机删一半（避免攻击者刷量把正常用户的验证码随机挤掉形成登录DoS）
	if len(s.items) >= s.maxItems {
		s.evictOldestLocked(s.maxItems / 10)
	}

	s.items[id] = &captchaItem{
		Code:      code,
		CreatedAt: time.Now(),
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

	if time.Now().After(item.ExpiresAt) {
		delete(s.items, id)
		return false
	}

	// 验证码不匹配：累计失败次数，达到上限立即作废
	if item.Code != answer {
		item.FailCount++
		if item.FailCount >= maxCaptchaFailCount {
			delete(s.items, id)
		}
		return false
	}

	// 验证通过立即删除，防止重放
	delete(s.items, id)
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

// evictOldestLocked 按创建时间淘汰最旧的 n 个验证码（在锁内调用）
func (s *captchaStore) evictOldestLocked(n int) {
	type kv struct {
		id string
		t  time.Time
	}
	list := make([]kv, 0, len(s.items))
	for id, item := range s.items {
		list = append(list, kv{id, item.CreatedAt})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].t.Before(list[j].t) })
	for i := 0; i < n && i < len(list); i++ {
		delete(s.items, list[i].id)
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

// cryptoRandInt 使用加密安全随机数生成 [0, max) 的随机整数
func cryptoRandInt(max int) int {
	n, err := rand.Int(rand.Reader, big.NewInt(int64(max)))
	if err != nil {
		return 0
	}
	return int(n.Int64())
}

// randDigitCode 生成随机数字验证码
func randDigitCode(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = byte('0' + cryptoRandInt(10))
	}
	return string(b)
}

// drawDigitCaptcha 绘制数字验证码图片
func drawDigitCaptcha(code string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))

	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	for i := 0; i < 6; i++ {
		drawNoiseLine(img)
	}

	for i := 0; i < 60; i++ {
		drawNoisePoint(img)
	}

	charWidth := captchaWidth / len(code)
	for i, ch := range code {
		x := i*charWidth + charWidth/4 + cryptoRandInt(6)
		y := captchaHeight/2 + 5 + cryptoRandInt(4) - 2
		drawDigit(img, int(ch-'0'), x, y, i)
	}

	return img
}

// drawDigit 绘制单个数字（使用简单可靠的实心方块字体）
func drawDigit(img *image.RGBA, digit, x, y, idx int) {
	col := fontColors[idx%len(fontColors)]
	// 3x5 点阵数字
	bitmap := digitBitmap[digit]
	pixelSize := 3 + cryptoRandInt(2) // 3~4 像素
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
	c := noiseColors[cryptoRandInt(len(noiseColors))]
	x1 := cryptoRandInt(captchaWidth)
	y1 := cryptoRandInt(captchaHeight)
	x2 := cryptoRandInt(captchaWidth)
	y2 := cryptoRandInt(captchaHeight)

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
	c := noiseColors[cryptoRandInt(len(noiseColors))]
	x := cryptoRandInt(captchaWidth)
	y := cryptoRandInt(captchaHeight)
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
