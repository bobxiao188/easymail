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

package sqlite

import (
	"path/filepath"
	"testing"
)

func TestNewPool(t *testing.T) {
	p := NewPool(Config{})
	if p == nil {
		t.Fatal("NewPool() returned nil")
	}
}

func TestPool_DB(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pool_test.db")
	p := NewPool(Config{})
	t.Cleanup(func() { _ = p.Close() })

	// First open should succeed
	db, err := p.DB(path)
	if err != nil {
		t.Fatalf("first DB() error = %v", err)
	}
	if db == nil {
		t.Fatal("first DB() returned nil")
	}

	// Second open with same path should return cached connection
	db2, err := p.DB(path)
	if err != nil {
		t.Fatalf("second DB() error = %v", err)
	}
	_ = db2

	// Different path should open a new connection
	path2 := filepath.Join(dir, "pool_test2.db")
	db3, err := p.DB(path2)
	if err != nil {
		t.Fatalf("third DB() error = %v", err)
	}
	if db3 == nil {
		t.Fatal("third DB() returned nil")
	}
}

func TestPool_DBEmptyPath(t *testing.T) {
	p := NewPool(Config{})
	_, err := p.DB("")
	if err == nil {
		t.Error("expected error for empty path")
	}
}

func TestPool_Close(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "close_test.db")
	p := NewPool(Config{})
	t.Cleanup(func() { _ = p.Close() })

	// Open a connection
	_, err := p.DB(path)
	if err != nil {
		t.Fatalf("DB() error = %v", err)
	}

	// Close all connections
	err = p.Close()
	if err != nil {
		t.Fatalf("Close() error = %v", err)
	}

	// After close, should be able to open again
	db, err := p.DB(path)
	if err != nil {
		t.Fatalf("DB() after close error = %v", err)
	}
	if db == nil {
		t.Fatal("DB() after close returned nil")
	}
}

func TestPool_CloseEmpty(t *testing.T) {
	p := NewPool(Config{})
	// Close with no connections should be safe
	err := p.Close()
	if err != nil {
		t.Errorf("Close() on empty pool error = %v", err)
	}
}
