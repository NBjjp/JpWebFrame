package token

import (
	"errors"
	"github.com/NBjjp/JpWebFrame"
	"github.com/golang-jwt/jwt/v4"
	"time"
)

const JWTToken = "jp_token"

type JwtHandler struct {
	//JWT算法
	Alg string
	//登录认证的方式
	Authenticator func(ctx *frame.Context) (map[string]any, error)
	//过期时间 秒
	TimeOut        time.Duration
	RefreshTimeOut time.Duration
	//时间函数 从此时开始计算过期
	TimeFuc func() time.Time
	//key
	Key []byte
	//刷新KEY
	RefreshKey []byte
	//私钥
	PrivateKey     string
	SendCookie     bool
	CookieName     string
	CookieMaxAge   int
	CookieDomain   string
	SecureCookie   bool
	CookieHTTPOnly bool
}

type JwtResponse struct {
	//返回给客户端
	Token string
	//存储在服务器中
	RefreshToken string
}

//登录过程 用户认证（用户密码）——>得到用户id   使用id生成Jwt 返回给客户端

func (j *JwtHandler) LoginHandler(ctx *frame.Context) (*JwtResponse, error) {
	//用户进行登录认证  data为用户id
	data, err := j.Authenticator(ctx)
	if err != nil {
		return nil, err
	}
	if j.Alg == "" {
		j.Alg = "HS256"
	}
	//处理A部分
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)
	//JWT b部分
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for key, value := range data {
			claims[key] = value
		}
	}
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.TimeOut)
	//过期时间
	claims["exp"] = expire.Unix()
	//发布时间
	claims["iat"] = j.TimeFuc().Unix()
	//C部分  生成token
	var tokenString string
	var tokenErr error
	if j.usingPublicKeyAlgo() {
		tokenString, tokenErr = token.SignedString(j.PrivateKey)
	} else {
		tokenString, tokenErr = token.SignedString(j.Key)
	}
	if tokenErr != nil {
		return nil, tokenErr
	}
	jr := &JwtResponse{
		Token: tokenString,
	}
	//refreshtoken
	refreshToken, err := j.refreshToken(data)
	if err != nil {
		return nil, err
	}
	jr.RefreshToken = refreshToken
	//发送存储cookie
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = int(expire.Unix() - j.TimeFuc().Unix())
		}
		maxAge := j.CookieMaxAge
		ctx.SetCookie(j.CookieName, tokenString, maxAge, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}

	return jr, nil
}

func (j *JwtHandler) usingPublicKeyAlgo() bool {
	switch j.Alg {
	case "RS256", "RS512", "RS384":
		return true
	}
	return false
}

func (j *JwtHandler) refreshToken(data map[string]any) (string, error) {
	signingMethod := jwt.GetSigningMethod(j.Alg)
	token := jwt.New(signingMethod)
	claims := token.Claims.(jwt.MapClaims)
	if data != nil {
		for key, value := range data {
			claims[key] = value
		}
	}
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.RefreshTimeOut)
	claims["exp"] = expire.Unix()
	claims["iat"] = j.TimeFuc().Unix()
	var tokenString string
	var errToken error
	if j.usingPublicKeyAlgo() {
		tokenString, errToken = token.SignedString(j.PrivateKey)
	} else {
		tokenString, errToken = token.SignedString(j.Key)
	}
	if errToken != nil {
		return "", errToken
	}
	return tokenString, nil
}

//退出登录
func (j *JwtHandler) LogoutHandler(ctx *frame.Context) error {
	//清除cookie即可
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		ctx.SetCookie(j.CookieName, "", -1, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
		return nil
	}
	return nil
}

func (j *JwtHandler) RefreshHandler(ctx *frame.Context) (*JwtResponse, error) {
	var token string
	//检测refresh token是否过期
	storageToken, exists := ctx.Get(string(j.RefreshKey))
	if exists {
		token = storageToken.(string)
	}
	if token == "" {
		return nil, errors.New("token not exist")
	}
	//解析Token
	t, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if j.usingPublicKeyAlgo() {
			return []byte(j.PrivateKey), nil
		}
		return []byte(j.Key), nil
	})
	if err != nil {
		return nil, err
	}
	claims := t.Claims.(jwt.MapClaims)
	//未过期的情况下 重新生成token和refreshToken
	if j.TimeFuc == nil {
		j.TimeFuc = func() time.Time {
			return time.Now()
		}
	}
	expire := j.TimeFuc().Add(j.RefreshTimeOut)
	claims["exp"] = expire.Unix()
	claims["iat"] = j.TimeFuc().Unix()

	var tokenString string
	var errToken error
	if j.usingPublicKeyAlgo() {
		tokenString, errToken = t.SignedString(j.PrivateKey)
	} else {
		tokenString, errToken = t.SignedString(j.Key)
	}
	if errToken != nil {
		return nil, errToken
	}
	if j.SendCookie {
		if j.CookieName == "" {
			j.CookieName = JWTToken
		}
		if j.CookieMaxAge == 0 {
			j.CookieMaxAge = int(expire.Unix() - j.TimeFuc().Unix())
		}
		maxAge := j.CookieMaxAge
		ctx.SetCookie(j.CookieName, tokenString, maxAge, "/", j.CookieDomain, j.SecureCookie, j.CookieHTTPOnly)
	}
	jr := &JwtResponse{}
	refreshToken, err := j.refreshToken(claims)
	if err != nil {
		return nil, err
	}
	jr.Token = tokenString
	jr.RefreshToken = refreshToken
	return jr, nil
}
