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
	"testing"
	"time"
)

func TestGenerateToken_Success(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	tok, err := GenerateToken("user-1", "alice", false, secret, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if tok == "" {
		t.Fatal("GenerateToken() returned empty token")
	}
}

func TestGenerateToken_IsAdmin(t *testing.T) {
	secret := "admin-secret-key-32-chars-minimum!!"
	tok, err := GenerateToken("admin-1", "admin", true, secret, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if !claims.IsAdmin {
		t.Error("admin token should have IsAdmin=true")
	}
}

func TestParseToken_Valid(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	tok, _ := GenerateToken("user-1", "alice", false, secret, 1)
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	if claims.UserID != "user-1" {
		t.Errorf("UserID = %q, want user-1", claims.UserID)
	}
	if claims.Username != "alice" {
		t.Errorf("Username = %q, want alice", claims.Username)
	}
	if claims.IsAdmin {
		t.Error("non-admin token should have IsAdmin=false")
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	tok, _ := GenerateToken("user-1", "alice", false, secret, 1)
	_, err := ParseToken(tok, "wrong-secret-key-32-chars-here!!!!")
	if err == nil {
		t.Fatal("ParseToken() with wrong secret should error")
	}
	if err != ErrInvalidToken {
		t.Errorf("ParseToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseToken_Expired(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	tok, _ := GenerateToken("user-1", "alice", false, secret, -1)
	_, err := ParseToken(tok, secret)
	if err == nil {
		t.Fatal("ParseToken() with expired token should error")
	}
	if err != ErrExpiredToken {
		t.Errorf("ParseToken() error = %v, want ErrExpiredToken", err)
	}
}

func TestParseToken_BadFormat(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	_, err := ParseToken("not.a.valid.token", secret)
	if err == nil {
		t.Fatal("ParseToken() with bad format should error")
	}
	if err != ErrInvalidToken {
		t.Errorf("ParseToken() error = %v, want ErrInvalidToken", err)
	}
}

func TestParseToken_Empty(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	_, err := ParseToken("", secret)
	if err == nil {
		t.Fatal("ParseToken() with empty token should error")
	}
}

func TestParseToken_CrossSecret(t *testing.T) {
	adminSecret := "admin-secret-key-32-chars-aaaaaaa!!"
	webmailSecret := "webmail-secret-key-32-chars-bbbbbb!!"
	tok, _ := GenerateToken("admin-1", "admin", true, adminSecret, 1)
	_, err := ParseToken(tok, webmailSecret)
	if err == nil {
		t.Fatal("admin token should not validate with webmail secret")
	}
}

func TestParseToken_AudienceCheck(t *testing.T) {
	secret := "webmail-secret-key-32-chars-ccc!!"
	tok, err := GenerateToken("user-2", "bob", false, secret, 1)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	okAud := false
	for _, a := range claims.Audience {
		if a == AdminAudience {
			okAud = true
			break
		}
	}
	if !okAud {
		t.Error("audience should contain admin")
	}
}

func TestClaims_ExpiryRange(t *testing.T) {
	secret := "test-secret-key-32-chars-minimum!!"
	expireHours := 2
	tok, _ := GenerateToken("user-1", "alice", false, secret, expireHours)
	claims, err := ParseToken(tok, secret)
	if err != nil {
		t.Fatalf("ParseToken() error = %v", err)
	}
	expectedMin := time.Now().Add(time.Duration(expireHours-1) * time.Hour)
	expectedMax := time.Now().Add(time.Duration(expireHours+1) * time.Hour)
	if claims.ExpiresAt.Time.Before(expectedMin) {
		t.Errorf("ExpiresAt %v should be after %v", claims.ExpiresAt.Time, expectedMin)
	}
	if claims.ExpiresAt.Time.After(expectedMax) {
		t.Errorf("ExpiresAt %v should be before %v", claims.ExpiresAt.Time, expectedMax)
	}
}

func TestConstants(t *testing.T) {
	if AdminAudience != "admin" {
		t.Errorf("AdminAudience = %q, want admin", AdminAudience)
	}
	if ErrInvalidToken.Error() == "" {
		t.Error("ErrInvalidToken should have non-empty message")
	}
	if ErrExpiredToken.Error() == "" {
		t.Error("ErrExpiredToken should have non-empty message")
	}
}
