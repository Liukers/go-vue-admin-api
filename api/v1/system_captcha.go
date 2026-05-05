package v1

import (
	"go-vue-admin/models/res"
	"go-vue-admin/util"

	"github.com/gin-gonic/gin"
)

type SystemCaptchaApi struct{}

// GetCaptcha
// @Tags 系统管理-认证
// @Summary 获取验证码
// @Description 获取图片验证码，返回验证码ID和base64图片
// @Produce json
// @Success 200 {object} res.Response{data=map[string]string} "成功"
// @Router /api/v1/system/captcha [get]
func (a *SystemCaptchaApi) GetCaptcha(c *gin.Context) {
	id, _, base64Img := util.GenerateCaptcha()
	res.Success(c, map[string]string{
		"captchaId": id,
		"captchaImg": base64Img,
	})
}
