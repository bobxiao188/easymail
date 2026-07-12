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

package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"

	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

type rateLimitEntry struct {
	count    int
	firstHit time.Time
}

// RateLimit returns a simple in-memory rate-limiter middleware.
// window defines the sliding window duration; maxRequests caps allowed requests per IP+path within that window.
func RateLimit(window time.Duration, maxRequests int) gin.HandlerFunc {
	var mu sync.Mutex
	buckets := make(map[string]*rateLimitEntry)

	// Periodically clean expired entries.
	go func() {
		tk := time.NewTicker(window)
		defer tk.Stop()
		for range tk.C {
			mu.Lock()
			now := time.Now()
			for k, e := range buckets {
				if now.Sub(e.firstHit) > window {
					delete(buckets, k)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := clientIP(c)
		key := ip + "|" + c.FullPath()

		mu.Lock()
		defer mu.Unlock()

		now := time.Now()
		e, ok := buckets[key]
		if !ok || now.Sub(e.firstHit) > window {
			buckets[key] = &rateLimitEntry{count: 1, firstHit: now}
			c.Next()
			return
		}

		e.count++
		if e.count > maxRequests {
			response.ErrorWithStatus(c, http.StatusTooManyRequests, 429, "Too many requests, please try again later.")
			c.Abort()
			return
		}
		c.Next()
	}
}

func clientIP(c *gin.Context) string {
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}
	host, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		return c.Request.RemoteAddr
	}
	return host
}
