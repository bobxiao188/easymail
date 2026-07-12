/*
 * Copyright (c) 2026 easymail.my. All rights reserved.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU Affero General Public License as published
 * by the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
 * GNU Affero General Public License for more details.
 *
 * Author: bob.xiao
 * License: AGPLv3
 */

package webmail

import (
	"context"

	"easymail/internal/domain/shared"
)

// ContactListResponse 联系人列表响应（带分页）
type ContactListResponse struct {
	Items   []ContactDTO `json:"items"`
	Total   int64        `json:"total"`
	Page    int          `json:"page"`
	PageSize int        `json:"page_size"`
}

// ContactService handles contact groups and contacts.
type ContactService interface {
	ListGroups(ctx context.Context, userID shared.GlobalID) ([]ContactGroupDTO, error)
	GetGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID) (*ContactGroupDTO, error)
	CreateGroup(ctx context.Context, userID shared.GlobalID, name string) (*ContactGroupDTO, error)
	UpdateGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID, name string) error
	DeleteGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID) error

	ListContacts(ctx context.Context, userID shared.GlobalID, query ListContactsQuery) (*ContactListResponse, error)
	GetContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID) (*ContactDTO, error)
	CreateContact(ctx context.Context, userID shared.GlobalID, input ContactInput) (*ContactDTO, error)
	UpdateContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID, input ContactInput) error
	DeleteContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID) error
}

// ---- DTOs ----

type ContactGroupDTO struct {
	ID           shared.GlobalID `json:"id"`
	GroupName    string          `json:"groupName"`
	IsDefault    bool            `json:"isDefault"`
	ContactCount int             `json:"contactCount"`
	CreateTime   string          `json:"createTime"`
}

type ContactDTO struct {
	ID             shared.GlobalID  `json:"id"`
	ContactName    string           `json:"contactName"`
	ContactEmail   string           `json:"contactEmail"`
	ContactPhone   string           `json:"contactPhone"`
	ContactAddress string           `json:"contactAddress"`
	ContactCity    string           `json:"contactCity"`
	ContactState   string           `json:"contactState"`
	ContactZip     string           `json:"contactZip"`
	ContactCountry string           `json:"contactCountry"`
	ContactGroupID *shared.GlobalID `json:"contactGroupId"`
}

type ContactInput struct {
	ContactName    string           `json:"contactName"`
	ContactEmail   string           `json:"contactEmail"`
	ContactPhone   string           `json:"contactPhone"`
	ContactAddress string           `json:"contactAddress"`
	ContactCity    string           `json:"contactCity"`
	ContactState   string           `json:"contactState"`
	ContactZip     string           `json:"contactZip"`
	ContactCountry string           `json:"contactCountry"`
	ContactGroupID *shared.GlobalID `json:"contactGroupId"`
}

type ListContactsQuery struct {
	Keyword   string
	GroupID   *shared.GlobalID
	Ungrouped bool
	Page      int
	PageSize  int
}

var _ ContactService = (*contactServiceImpl)(nil)
