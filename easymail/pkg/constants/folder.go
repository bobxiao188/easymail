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

package constants

// FolderID 逻辑文件夹枚举（MySQL folder_kind、SQLite folder 引用一致）
// 文件夹 ID 从 1 开始：Inbox=1, Sent=2, Draft=3, Trash=4, Spam=5, Quarantine=6
// 用户自建文件夹从 100 开始
type FolderID int64

const (
	_          FolderID = 0 // 占位，从 1 开始
	Inbox      FolderID = 1 // 1 - 收件箱
	Sent       FolderID = 2 // 2 - 发件箱
	Draft      FolderID = 3 // 3 - 草稿
	Trash      FolderID = 4 // 4 - 已删除
	Spam       FolderID = 5 // 5 - 垃圾邮件
	Quarantine FolderID = 6 // 6 - 隔离
	// FolderUserCustomMin 及以上为用户自建文件夹（与系统内 1..Quarantine 区分）
	FolderUserCustomMin FolderID = 100
)

// FolderKey 机器可读键，用于配置、API、种子数据
type FolderKey string

const (
	FolderKeyInbox      FolderKey = "inbox"
	FolderKeySent       FolderKey = "sent"
	FolderKeyDraft      FolderKey = "draft"
	FolderKeyTrash      FolderKey = "trash"
	FolderKeySpam       FolderKey = "spam"
	FolderKeyQuarantine FolderKey = "quarantine"
)

// Key 返回默认文件夹键
func (k FolderID) Key() FolderKey {
	switch k {
	case Inbox:
		return FolderKeyInbox
	case Sent:
		return FolderKeySent
	case Draft:
		return FolderKeyDraft
	case Trash:
		return FolderKeyTrash
	case Spam:
		return FolderKeySpam
	case Quarantine:
		return FolderKeyQuarantine
	default:
		return ""
	}
}

// DefaultSeedFolderKinds 新建邮箱时默认创建的系统文件夹（一级目录）
func DefaultSeedFolderKinds() []FolderID {
	return []FolderID{Inbox, Sent, Draft, Trash, Spam, Quarantine}
}

// DefaultFolderDisplayName 默认展示名（中文），供管理端、Webmail 使用
// IsSystemFolderKind 系统默认文件夹（不可删除、不可重命名）
func IsSystemFolderKind(kind FolderID) bool {
	return kind >= Inbox && kind <= Quarantine
}

func DefaultFolderDisplayName(kind FolderID) string {
	switch kind {
	case Inbox:
		return "收件箱"
	case Sent:
		return "发件箱"
	case Draft:
		return "草稿"
	case Trash:
		return "已删除"
	case Spam:
		return "垃圾邮件"
	case Quarantine:
		return "隔离区"
	default:
		return "文件"
	}
}
