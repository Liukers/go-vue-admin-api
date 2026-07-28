package util

import (
	"strings"
)

// ParseUserAgent 解析 User-Agent 获取浏览器和操作系统信息
func ParseUserAgent(userAgent string) (browser, os string) {
	if userAgent == "" {
		return "Unknown", "Unknown"
	}

	browser = parseBrowser(userAgent)
	os = parseOS(userAgent)

	return browser, os
}

// parseBrowser 解析浏览器
func parseBrowser(userAgent string) string {
	userAgent = strings.ToLower(userAgent)

	if strings.Contains(userAgent, "edg/") || strings.Contains(userAgent, "edge/") {
		return "Microsoft Edge"
	}

	// Chrome 需要在 Safari 之前检查，因为 Chrome 也包含 Safari
	if strings.Contains(userAgent, "chrome/") && !strings.Contains(userAgent, "chromium/") {
		return "Chrome"
	}

	if strings.Contains(userAgent, "firefox/") {
		return "Firefox"
	}

	// Safari 需要在 Chrome 之后检查
	if strings.Contains(userAgent, "safari/") && !strings.Contains(userAgent, "chrome/") {
		return "Safari"
	}

	if strings.Contains(userAgent, "opr/") || strings.Contains(userAgent, "opera/") {
		return "Opera"
	}

	if strings.Contains(userAgent, "trident/") || strings.Contains(userAgent, "msie ") {
		return "Internet Explorer"
	}

	if strings.Contains(userAgent, "micromessenger/") {
		return "WeChat"
	}

	if strings.Contains(userAgent, "mobile/") {
		return "Mobile Browser"
	}

	return "Unknown"
}

// parseOS 解析操作系统
func parseOS(userAgent string) string {
	userAgent = strings.ToLower(userAgent)

	if strings.Contains(userAgent, "windows nt 10.0") {
		return "Windows 10/11"
	}
	if strings.Contains(userAgent, "windows nt 6.3") {
		return "Windows 8.1"
	}
	if strings.Contains(userAgent, "windows nt 6.2") {
		return "Windows 8"
	}
	if strings.Contains(userAgent, "windows nt 6.1") {
		return "Windows 7"
	}
	if strings.Contains(userAgent, "windows") {
		return "Windows"
	}

	if strings.Contains(userAgent, "macintosh") || strings.Contains(userAgent, "mac os x") {
		return "macOS"
	}

	if strings.Contains(userAgent, "iphone") {
		return "iOS (iPhone)"
	}
	if strings.Contains(userAgent, "ipad") {
		return "iOS (iPad)"
	}

	if strings.Contains(userAgent, "android") {
		return "Android"
	}

	if strings.Contains(userAgent, "linux") {
		return "Linux"
	}

	if strings.Contains(userAgent, "ubuntu") {
		return "Ubuntu"
	}

	return "Unknown"
}

// GetIPLocation 根据 IP 获取地理位置（简化版）
func GetIPLocation(ip string) string {
	if ip == "127.0.0.1" || ip == "::1" || ip == "localhost" {
		return "本机地址"
	}

	if strings.HasPrefix(ip, "192.168.") ||
		strings.HasPrefix(ip, "10.") ||
		strings.HasPrefix(ip, "172.16.") ||
		strings.HasPrefix(ip, "172.17.") ||
		strings.HasPrefix(ip, "172.18.") ||
		strings.HasPrefix(ip, "172.19.") ||
		strings.HasPrefix(ip, "172.20.") ||
		strings.HasPrefix(ip, "172.21.") ||
		strings.HasPrefix(ip, "172.22.") ||
		strings.HasPrefix(ip, "172.23.") ||
		strings.HasPrefix(ip, "172.24.") ||
		strings.HasPrefix(ip, "172.25.") ||
		strings.HasPrefix(ip, "172.26.") ||
		strings.HasPrefix(ip, "172.27.") ||
		strings.HasPrefix(ip, "172.28.") ||
		strings.HasPrefix(ip, "172.29.") ||
		strings.HasPrefix(ip, "172.30.") ||
		strings.HasPrefix(ip, "172.31.") {
		return "局域网"
	}

	// 这里可以集成第三方 IP 地理位置服务
	// 如：ip2region、淘宝 IP 库、百度地图 API 等
	return "未知"
}
