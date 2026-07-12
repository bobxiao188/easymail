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

package easydns

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestNewResolver_Default(t *testing.T) {
	r := CreateDefaultResolver()
	if r == nil {
		t.Fatal("CreateDefaultResolver() returned nil")
	}
	s := r.String()
	if !strings.Contains(s, "dns") && !strings.Contains(s, "resolver") {
		t.Logf("CreateDefaultResolver().String() = %q", s)
	}
}

func TestNewResolver_Custom(t *testing.T) {
	r := NewResolver(Config{
		Servers: []string{"8.8.8.8:53"},
	})
	if r == nil {
		t.Fatal("NewResolver() returned nil")
	}
	s := r.String()
	if !strings.Contains(s, "8.8.8.8") {
		t.Errorf("String() = %q, want to contain '8.8.8.8'", s)
	}
}

func TestNewResolver_EmptyAddr(t *testing.T) {
	r := NewResolver(Config{})
	if r == nil {
		t.Fatal("NewResolver(Config{}) returned nil")
	}
}

func TestNewResolver_IPv6(t *testing.T) {
	r := NewResolver(Config{
		Servers: []string{"[2001:4860:4860::8888]:53"},
	})
	if r == nil {
		t.Fatal("NewResolver() returned nil")
	}
	s := r.String()
	if s == "" {
		t.Error("String() returned empty")
	}
}

func TestResolver_String(t *testing.T) {
	r := NewResolver(Config{
		Servers: []string{"1.1.1.1:53"},
	})
	s := r.String()
	if s == "" {
		t.Error("String() should not be empty")
	}
}

func TestResolver_Concurrent(t *testing.T) {
	r := NewResolver(Config{
		Servers:    []string{"8.8.8.8:53", "1.1.1.1:53"},
		MaxRetries: 2,
		Timeout:    5 * time.Second,
	})

	var wg sync.WaitGroup
	errors := make(chan error, 100)

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := r.LookupTXT(context.Background(), "google.com")
			if err != nil {
				errors <- err
			}
		}()
	}

	wg.Wait()
	close(errors)

	errCount := 0
	for err := range errors {
		t.Logf("Error: %v", err)
		errCount++
	}

	if errCount > 10 {
		t.Errorf("Too many errors: %d", errCount)
	}
}

func TestResolver_SetServers(t *testing.T) {
	r := NewResolver(Config{
		Servers: []string{"8.8.8.8:53"},
	})

	r.SetServers([]string{"1.1.1.1:53", "8.8.4.4:53"})
	servers := r.GetServers()
	if len(servers) != 2 {
		t.Errorf("Expected 2 servers, got %d", len(servers))
	}
}

func TestResolver_SetTimeout(t *testing.T) {
	r := NewResolver(Config{
		Timeout: 5 * time.Second,
	})

	r.SetTimeout(10 * time.Second)
	timeout := r.GetTimeout()
	if timeout != 10*time.Second {
		t.Errorf("Expected timeout 10s, got %v", timeout)
	}
}

func TestResolver_IsHealthy(t *testing.T) {
	r := NewResolver(Config{
		Servers: []string{"8.8.8.8:53"},
		Timeout: 2 * time.Second,
	})

	if !r.IsHealthy() {
		t.Error("Expected resolver to be healthy")
	}
}

func TestDefaultResolver(t *testing.T) {
	SetDefault(nil)
	r := GetDefault()
	if r == nil {
		t.Fatal("GetDefault() returned nil after SetDefault(nil)")
	}
}
