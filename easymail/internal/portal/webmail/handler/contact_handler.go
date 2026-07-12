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

package handler

import (
	"errors"
	"strconv"
	"strings"

	"easymail/internal/app/webmail"
	"easymail/internal/domain/shared"
	"easymail/internal/portal/webmail/middleware"
	appi18n "easymail/pkg/i18n"
	"easymail/pkg/response"

	"github.com/gin-gonic/gin"
)

type contactGroupReq struct {
	GroupName string `json:"groupName" binding:"required"`
}

func contactGroupToResp(c *gin.Context, g *webmail.ContactGroupDTO) gin.H {
	groupName := g.GroupName
	// Localize default group name based on request language
	if g.IsDefault {
		groupName = appi18n.Message(c, appi18n.KeyContactGroupDefault)
	}
	return gin.H{
		"id": g.ID, "groupName": groupName,
		"isDefault": g.IsDefault, "contactCount": g.ContactCount,
		"createTime": g.CreateTime,
	}
}

func contactGroupsToListResp(c *gin.Context, list []webmail.ContactGroupDTO) []gin.H {
	out := make([]gin.H, len(list))
	for i := range list {
		out[i] = contactGroupToResp(c, &list[i])
	}
	return out
}

// ListContactGroups returns all contact groups for the account.
func (h *Handler) ListContactGroups(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	list, err := h.contactService.ListGroups(c.Request.Context(), aid)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, contactGroupsToListResp(c, list))
}

// CreateContactGroup creates a new contact group.
func (h *Handler) CreateContactGroup(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req contactGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	g, err := h.contactService.CreateGroup(c.Request.Context(), aid, req.GroupName)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, contactGroupToResp(c, g))
}

// UpdateContactGroup renames a contact group.
func (h *Handler) UpdateContactGroup(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	gidStr := c.Param("id")
	if gidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroupID))
		return
	}
	gid := shared.GlobalID(gidStr)

	// Check if it is the default group
	group, err := h.contactService.GetGroup(c.Request.Context(), aid, gid)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	if group.IsDefault {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsDefaultGroupCannotModify))
		return
	}

	var req contactGroupReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	if err := h.contactService.UpdateGroup(c.Request.Context(), aid, gid, req.GroupName); err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DeleteContactGroup removes a group (contacts may be ungrouped per service rules).
func (h *Handler) DeleteContactGroup(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	gidStr := c.Param("id")
	if gidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroupID))
		return
	}
	gid := shared.GlobalID(gidStr)

	// Check if it is the default group
	group, err := h.contactService.GetGroup(c.Request.Context(), aid, gid)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	if group.IsDefault {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsDefaultGroupCannotDelete))
		return
	}

	if err := h.contactService.DeleteGroup(c.Request.Context(), aid, gid); err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

type contactCreateReq struct {
	ContactName    string `json:"contactName" binding:"required"`
	ContactEmail   string `json:"contactEmail" binding:"required"`
	ContactPhone   string `json:"contactPhone"`
	ContactAddress string `json:"contactAddress"`
	ContactCity    string `json:"contactCity"`
	ContactState   string `json:"contactState"`
	ContactZip     string `json:"contactZip"`
	ContactCountry string `json:"contactCountry"`
	ContactGroupID *shared.GlobalID `json:"contactGroupId"`
}

// ListContacts returns contacts with optional group or ungrouped filter.
func (h *Handler) ListContacts(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	
	q := webmail.ListContactsQuery{
		Keyword:  c.Query("q"),
		Page:     page,
		PageSize: pageSize,
	}
	if c.Query("ungrouped") == "1" || strings.EqualFold(c.Query("ungrouped"), "true") {
		q.Ungrouped = true
	} else if v := strings.TrimSpace(c.Query("group_id")); v != "" {
		gid := shared.GlobalID(v)
		q.GroupID = &gid
	}
	
	result, err := h.contactService.ListContacts(c.Request.Context(), aid, q)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, result)
}

// CreateContact adds a contact.
func (h *Handler) CreateContact(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	var req contactCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	in := webmail.ContactInput{
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		ContactAddress: req.ContactAddress,
		ContactCity:    req.ContactCity,
		ContactState:   req.ContactState,
		ContactZip:     req.ContactZip,
		ContactCountry: req.ContactCountry,
		ContactGroupID: normalizeGroupID(req.ContactGroupID),
	}
	co, err := h.contactService.CreateContact(c.Request.Context(), aid, in)
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, co)
}

// UpdateContact updates an existing contact.
func (h *Handler) UpdateContact(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	cidStr := c.Param("id")
	if cidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidContactID))
		return
	}
	cid := shared.GlobalID(cidStr)
	var req contactCreateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyErrBadRequest))
		return
	}
	in := webmail.ContactInput{
		ContactName:    req.ContactName,
		ContactEmail:   req.ContactEmail,
		ContactPhone:   req.ContactPhone,
		ContactAddress: req.ContactAddress,
		ContactCity:    req.ContactCity,
		ContactState:   req.ContactState,
		ContactZip:     req.ContactZip,
		ContactCountry: req.ContactCountry,
		ContactGroupID: normalizeGroupID(req.ContactGroupID),
	}
	if err := h.contactService.UpdateContact(c.Request.Context(), aid, cid, in); err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// DeleteContact removes a contact.
func (h *Handler) DeleteContact(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	cidStr := c.Param("id")
	if cidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidContactID))
		return
	}
	cid := shared.GlobalID(cidStr)
	if err := h.contactService.DeleteContact(c.Request.Context(), aid, cid); err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, gin.H{"ok": true})
}

// GetContact returns a single contact by ID.
func (h *Handler) GetContact(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	cidStr := c.Param("id")
	if cidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidContactID))
		return
	}
	co, err := h.contactService.GetContact(c.Request.Context(), aid, shared.GlobalID(cidStr))
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, co)
}

// GetContactGroup returns a single contact group by ID.
func (h *Handler) GetContactGroup(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	gidStr := c.Param("id")
	if gidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroupID))
		return
	}
	g, err := h.contactService.GetGroup(c.Request.Context(), aid, shared.GlobalID(gidStr))
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, contactGroupToResp(c, g))
}

// GetGroupContacts returns all contacts in a contact group.
func (h *Handler) GetGroupContacts(c *gin.Context) {
	if h.contactService == nil {
		response.InternalError(c, appi18n.Message(c, appi18n.KeyWebmailContactsUnavailable))
		return
	}
	aid, ok := middleware.MailUserID(c)
	if !ok || aid == "" {
		response.Unauthorized(c, "")
		return
	}
	gidStr := c.Param("id")
	if gidStr == "" {
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroupID))
		return
	}
	gid := shared.GlobalID(gidStr)
	result, err := h.contactService.ListContacts(c.Request.Context(), aid, webmail.ListContactsQuery{
		GroupID:  &gid,
		Page:     1,
		PageSize: 1000,
	})
	if err != nil {
		h.writeContactErr(c, err)
		return
	}
	response.Success(c, result)
}

func normalizeGroupID(p *shared.GlobalID) *shared.GlobalID {
	if p == nil || *p == "" {
		return nil
	}
	return p
}

func (h *Handler) writeContactErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, webmail.ErrNotFound):
		response.NotFound(c, appi18n.Message(c, appi18n.KeyWebmailContactsNotFound))
	case errors.Is(err, webmail.ErrDuplicate):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsDuplicate))
	case errors.Is(err, webmail.ErrInvalidEmail):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailInvalidEmail))
	case errors.Is(err, webmail.ErrInvalidGroup):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidGroup))
	case errors.Is(err, webmail.ErrInvalidArgument):
		response.BadRequest(c, appi18n.Message(c, appi18n.KeyWebmailContactsInvalidArgument))
	default:
		h.log.Errorf("webmail contact handler error: %+v", err)
		response.InternalError(c, appi18n.Message(c, appi18n.KeyErrInternalServer))
	}
}

