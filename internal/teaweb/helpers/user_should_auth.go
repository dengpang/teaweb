package helpers

import (
	"fmt"
	"github.com/iwind/TeaGo/actions"
	"net/http"
)

type UserShouldAuth struct {
	action *actions.ActionObject
}

func (this *UserShouldAuth) BeforeAction(actionPtr actions.ActionWrapper, paramName string) (goNext bool) {
	this.action = actionPtr.Object()

	// 安全
	action := this.action
	action.AddHeader("X-Frame-Options", "SAMEORIGIN")
	action.AddHeader("Content-Security-Policy", "default-src 'self' data:; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'")

	return true
}

// 存储用户名到SESSION
func (this *UserShouldAuth) StoreUsername(username string, remember bool) {
	// 修改sid的时间
	if remember {
		cookie := &http.Cookie{
			Name:     "sid",
			Value:    this.action.Session().Sid,
			Path:     "/",
			MaxAge:   14 * 86400,
			HttpOnly: true,
		}
		if this.action.Request.TLS != nil {
			cookie.SameSite = http.SameSiteStrictMode
			cookie.Secure = true
		}
		this.action.AddCookie(cookie)
	} else {
		cookie := &http.Cookie{
			Name:     "sid",
			Value:    this.action.Session().Sid,
			Path:     "/",
			MaxAge:   0,
			HttpOnly: true,
		}
		if this.action.Request.TLS != nil {
			cookie.SameSite = http.SameSiteStrictMode
			cookie.Secure = true
		}
		this.action.AddCookie(cookie)
	}
	res := this.action.Session().Write("username", username)
	fmt.Println("res==", res)
	fmt.Println(this.action.Session().GetString("username"))
}

func (this *UserShouldAuth) Logout() {
	this.action.Session().Delete()
}
