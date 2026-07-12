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

	domcontact "easymail/internal/domain/contact"
	"easymail/internal/domain/shared"
)

type contactServiceImpl struct {
	contactRepo domcontact.ContactRepository
	groupRepo   domcontact.ContactGroupRepository
}

// NewContactService creates a ContactService backed by domain-level repository interfaces.
func NewContactService(contactRepo domcontact.ContactRepository, groupRepo domcontact.ContactGroupRepository) ContactService {
	return &contactServiceImpl{contactRepo: contactRepo, groupRepo: groupRepo}
}

func (s *contactServiceImpl) ListGroups(ctx context.Context, userID shared.GlobalID) ([]ContactGroupDTO, error) {
	groups, err := s.groupRepo.ListByAccount(ctx, userID)
	if err != nil {
		return nil, err
	}
	return groupsToDTOsWithCount(ctx, groups, userID, s.contactRepo), nil
}

func (s *contactServiceImpl) GetGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID) (*ContactGroupDTO, error) {
	g, err := s.groupRepo.FindByAccountAndID(ctx, userID, groupID)
	if err != nil {
		return nil, mapContactError(err)
	}
	return groupToDTO(g), nil
}

func (s *contactServiceImpl) CreateGroup(ctx context.Context, userID shared.GlobalID, name string) (*ContactGroupDTO, error) {
	g, err := domcontact.NewContactGroup(userID, name)
	if err != nil {
		return nil, mapContactError(err)
	}
	if err := s.groupRepo.Save(ctx, g); err != nil {
		return nil, mapContactError(err)
	}
	return groupToDTO(g), nil
}

func (s *contactServiceImpl) UpdateGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID, name string) error {
	g, err := s.groupRepo.FindByAccountAndID(ctx, userID, groupID)
	if err != nil {
		return mapContactError(err)
	}
	if err := g.Rename(name); err != nil {
		return mapContactError(err)
	}
	return s.groupRepo.Save(ctx, g)
}

func (s *contactServiceImpl) DeleteGroup(ctx context.Context, userID shared.GlobalID, groupID shared.GlobalID) error {
	return mapContactError(s.groupRepo.Delete(ctx, userID, groupID))
}

func (s *contactServiceImpl) ListContacts(ctx context.Context, userID shared.GlobalID, query ListContactsQuery) (*ContactListResponse, error) {
	// 设置默认分页参数
	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}
	
	// 获取总数
	total, err := s.contactRepo.Count(ctx, userID, query.GroupID, query.Ungrouped)
	if err != nil {
		return nil, err
	}
	
	// 获取分页数据
	contacts, err := s.contactRepo.SearchPaged(ctx, userID, query.Keyword, query.GroupID, query.Ungrouped, query.Page, query.PageSize)
	if err != nil {
		return nil, err
	}
	
	return &ContactListResponse{
		Items:    contactsToDTOs(contacts),
		Total:    total,
		Page:     query.Page,
		PageSize: query.PageSize,
	}, nil
}

func (s *contactServiceImpl) CreateContact(ctx context.Context, userID shared.GlobalID, input ContactInput) (*ContactDTO, error) {
	c, err := domcontact.NewContact(userID, input.ContactName, input.ContactEmail)
	if err != nil {
		return nil, mapContactError(err)
	}
	c.ContactPhone = input.ContactPhone
	c.ContactAddress = input.ContactAddress
	c.ContactCity = input.ContactCity
	c.ContactState = input.ContactState
	c.ContactZip = input.ContactZip
	c.ContactCountry = input.ContactCountry
	c.ContactGroupID = input.ContactGroupID

	if err := s.contactRepo.Save(ctx, c); err != nil {
		return nil, mapContactError(err)
	}
	return contactToDTO(c), nil
}

func (s *contactServiceImpl) UpdateContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID, input ContactInput) error {
	c, err := s.contactRepo.FindByAccountAndID(ctx, userID, contactID)
	if err != nil {
		return mapContactError(err)
	}
	c.ContactName = input.ContactName
	if err := c.UpdateEmail(input.ContactEmail); err != nil {
		return mapContactError(err)
	}
	c.ContactPhone = input.ContactPhone
	c.ContactAddress = input.ContactAddress
	c.ContactCity = input.ContactCity
	c.ContactState = input.ContactState
	c.ContactZip = input.ContactZip
	c.ContactCountry = input.ContactCountry
	c.ContactGroupID = input.ContactGroupID

	return mapContactError(s.contactRepo.Save(ctx, c))
}

func (s *contactServiceImpl) GetContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID) (*ContactDTO, error) {
	c, err := s.contactRepo.FindByAccountAndID(ctx, userID, contactID)
	if err != nil {
		return nil, mapContactError(err)
	}
	return contactToDTO(c), nil
}

func (s *contactServiceImpl) DeleteContact(ctx context.Context, userID shared.GlobalID, contactID shared.GlobalID) error {
	return mapContactError(s.contactRepo.Delete(ctx, userID, contactID))
}

// ---- DTO converters ----

func groupToDTO(g *domcontact.ContactGroup) *ContactGroupDTO {
	if g == nil {
		return nil
	}
	return &ContactGroupDTO{
		ID:         g.ID,
		GroupName:  g.GroupName,
		IsDefault:  g.IsDefault,
		CreateTime: g.CreateTime.Format("2006-01-02 15:04:05"),
	}
}

func groupsToDTOs(list []domcontact.ContactGroup) []ContactGroupDTO {
	out := make([]ContactGroupDTO, len(list))
	for i := range list {
		out[i] = ContactGroupDTO{
			ID:         list[i].ID,
			GroupName:  list[i].GroupName,
			IsDefault:  list[i].IsDefault,
			CreateTime: list[i].CreateTime.Format("2006-01-02 15:04:05"),
		}
	}
	return out
}

// groupsToDTOsWithCount converts groups to DTOs with contact count for each group.
func groupsToDTOsWithCount(ctx context.Context, list []domcontact.ContactGroup, userID shared.GlobalID, repo domcontact.ContactRepository) []ContactGroupDTO {
	out := make([]ContactGroupDTO, len(list))
	for i := range list {
		var count int
		if c, err := repo.Count(ctx, userID, &list[i].ID, false); err == nil {
			count = int(c)
		}
		out[i] = ContactGroupDTO{
			ID:           list[i].ID,
			GroupName:    list[i].GroupName,
			IsDefault:    list[i].IsDefault,
			ContactCount: count,
			CreateTime:   list[i].CreateTime.Format("2006-01-02 15:04:05"),
		}
	}
	return out
}

func contactToDTO(c *domcontact.Contact) *ContactDTO {
	if c == nil {
		return nil
	}
	return &ContactDTO{
		ID:             c.ID,
		ContactName:    c.ContactName,
		ContactEmail:   c.ContactEmail,
		ContactPhone:   c.ContactPhone,
		ContactAddress: c.ContactAddress,
		ContactCity:    c.ContactCity,
		ContactState:   c.ContactState,
		ContactZip:     c.ContactZip,
		ContactCountry: c.ContactCountry,
		ContactGroupID: c.ContactGroupID,
	}
}

func contactsToDTOs(list []domcontact.Contact) []ContactDTO {
	out := make([]ContactDTO, len(list))
	for i := range list {
		out[i] = *contactToDTO(&list[i])
	}
	return out
}

func mapContactError(err error) error {
	if err == nil {
		return nil
	}
	switch err {
	case domcontact.ErrContactNotFound, domcontact.ErrGroupNotFound:
		return ErrNotFound
	case domcontact.ErrContactDuplicate, domcontact.ErrGroupDuplicate:
		return ErrDuplicate
	case domcontact.ErrContactInvalidEmail:
		return ErrInvalidEmail
	case domcontact.ErrGroupInvalidName, domcontact.ErrContactInvalidName:
		return ErrInvalidArgument
	default:
		return err
	}
}
