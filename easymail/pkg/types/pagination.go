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

package types

import "github.com/gin-gonic/gin"

type PaginationRequest struct {
	Page      int    `form:"page" json:"page"`
	PageSize  int    `form:"pageSize" json:"pageSize"`
	Keyword   string `form:"keyword" json:"keyword"`
	SortBy    string `form:"sortBy" json:"sortBy"`
	SortOrder string `form:"sortOrder" json:"sortOrder"`
	Search    string `form:"search" json:"search"`
}

func (p *PaginationRequest) Validate() error {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 10
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	if p.SortOrder != "asc" && p.SortOrder != "desc" {
		p.SortOrder = "desc"
	}
	return nil
}

func (p *PaginationRequest) Offset() int {
	return (p.Page - 1) * p.PageSize
}

func (p *PaginationRequest) Limit() int {
	return p.PageSize
}

func (p *PaginationRequest) FromGinContext(c *gin.Context) error {
	if err := c.ShouldBindQuery(p); err != nil {
		return err
	}
	return p.Validate()
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"pageSize"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"totalPages"`
	HasPrev    bool  `json:"hasPrev"`
	HasNext    bool  `json:"hasNext"`
	FirstPage  int   `json:"firstPage"`
	LastPage   int   `json:"lastPage"`
}

func NewPaginationMeta(page, pageSize int, total int64) PaginationMeta {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return PaginationMeta{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
		HasPrev:    page > 1,
		HasNext:    page < totalPages,
		FirstPage:  1,
		LastPage:   totalPages,
	}
}
