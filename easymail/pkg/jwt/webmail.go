/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * For commercial licensing inquiries, please contact: 3680010825@qq.com
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const WebmailAudience = "webmail"

// WebmailClaims 缁堢鐢ㄦ埛 Webmail JWT锛堜笌绠＄悊Claims 鍒嗙锛岄伩鍏嶆贩鐢級
type WebmailClaims struct {
	MailUserID string `json:"accountId"`
	Email      string `json:"email"`
	jwt.RegisteredClaims
}

// GenerateWebmailToken 绛惧彂 aud=webmail JWT
func GenerateWebmailToken(MailUserID string, email string, secret string, expireHours int) (string, error) {
	if expireHours <= 0 {
		expireHours = 24
	}
	expirationTime := time.Now().Add(time.Duration(expireHours) * time.Hour)
	claims := &WebmailClaims{
		MailUserID: MailUserID,
		Email:      email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Subject:   email,
			Audience:  jwt.ClaimStrings{WebmailAudience},
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

func ParseWebmailToken(tokenString string, secret string) (*WebmailClaims, error) {
	claims := &WebmailClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}
	if !token.Valid {
		return nil, ErrInvalidToken
	}
	okAud := false
	for _, a := range claims.Audience {
		if a == WebmailAudience {
			okAud = true
			break
		}
	}
	if !okAud {
		return nil, ErrInvalidToken
	}
	return claims, nil
}
