package captcha

import (
	"github.com/mojocn/base64Captcha"
)

var defaultStore = base64Captcha.DefaultMemStore

type CaptchaType string

const (
	String  CaptchaType = "string"
	Audio   CaptchaType = "audio"
	Math    CaptchaType = "math"
	Chinese CaptchaType = "chinese"
	Digit   CaptchaType = "digit"
)

func NewCaptchaDriverWithStore(captchaType CaptchaType, store base64Captcha.Store) *base64Captcha.Captcha {
	var driver base64Captcha.Driver
	switch captchaType {
	case String:
		bgColor := base64Captcha.RandLightColor()
		driver = base64Captcha.NewDriverString(60, 120, 0,
			base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSlimeLine,
			4, "23456789qwertyuipkjhgfdsazxcvbnm", &bgColor, nil,
			[]string{"wqy-microhei.ttc"},
		)
	case Audio:
		driver = base64Captcha.NewDriverAudio(6, "zh")
	case Math:
		bgColor := base64Captcha.RandLightColor()
		driver = base64Captcha.NewDriverMath(60, 120, 0,
			base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSlimeLine,
			&bgColor, nil,
			[]string{"wqy-microhei.ttc"})
	case Chinese:
		bgColor := base64Captcha.RandLightColor()
		driver = base64Captcha.NewDriverChinese(60, 120, 0,
			base64Captcha.OptionShowHollowLine|base64Captcha.OptionShowSlimeLine, 4,
			"设想,你在,处理,消费者,的音,频输,出音,频可,能无,论什,么都,没有,任何,输出,或者,它可,能是,单声道,立体声,或是,环绕立,体声的,,不想要,的值",
			&bgColor, nil,
			[]string{"wqy-microhei.ttc"})
	case Digit:
		fallthrough
	default:
		driver = base64Captcha.NewDriverDigit(60, 120, 5, 0.7, 80)
	}

	return base64Captcha.NewCaptcha(driver, store)
}

func NewCaptchaDriver(captchaType CaptchaType) *base64Captcha.Captcha {
	return NewCaptchaDriverWithStore(captchaType, defaultStore)
}
