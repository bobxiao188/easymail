package mysql

import (
	"context"
	"errors"
	"time"

	"easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/internal/infrastructure/persistence"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// --- PostfixAgent PO converters ---

func poToPostfixAgent(po *PostfixAgentPO) *management.PostfixAgent {
	if po == nil {
		return nil
	}
	return &management.PostfixAgent{
		ID:          po.ID,
		Name:        po.Name,
		Host:        po.Host,
		Token:       po.Token,
		Enabled:     po.Enabled,
		LastStatus:  po.LastStatus,
		LastSyncAt:  po.LastSyncAt,
		Description: po.Description,
		CreateTime:  po.CreateTime,
		UpdateTime:  po.UpdateTime,
	}
}

func postfixAgentToPO(a *management.PostfixAgent) *PostfixAgentPO {
	if a == nil {
		return nil
	}
	return &PostfixAgentPO{
		ID:          a.ID,
		Name:        a.Name,
		Host:        a.Host,
		Token:       a.Token,
		Enabled:     a.Enabled,
		LastStatus:  a.LastStatus,
		LastSyncAt:  a.LastSyncAt,
		Description: a.Description,
		CreateTime:  a.CreateTime,
		UpdateTime:  a.UpdateTime,
	}
}

// PostfixAgentRepository implements management.PostfixAgentRepository.
type PostfixAgentRepository struct {
	db persistence.DBProvider
}

func NewPostfixAgentRepository(db persistence.DBProvider) *PostfixAgentRepository {
	return &PostfixAgentRepository{db: db}
}

func (r *PostfixAgentRepository) Save(ctx context.Context, a *management.PostfixAgent) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	var exists int64
	if err := g.Model(&PostfixAgentPO{}).
		Where("name = ? AND id <> ?", a.Name, a.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return management.ErrPostfixAgentExists
	}
	po := postfixAgentToPO(a)
	po.UpdateTime = time.Now()
	return g.Save(po).Error
}

func (r *PostfixAgentRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.PostfixAgent, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po PostfixAgentPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrPostfixAgentNotFound
		}
		return nil, err
	}
	return poToPostfixAgent(&po), nil
}

func (r *PostfixAgentRepository) FindAll(ctx context.Context) ([]management.PostfixAgent, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []PostfixAgentPO
	if err := g.Order("name").Find(&pos).Error; err != nil {
		return nil, err
	}
	agents := make([]management.PostfixAgent, len(pos))
	for i := range pos {
		agents[i] = *poToPostfixAgent(&pos[i])
	}
	return agents, nil
}

func (r *PostfixAgentRepository) FindEnabled(ctx context.Context) ([]management.PostfixAgent, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []PostfixAgentPO
	if err := g.Where("enabled = ?", true).Order("name").Find(&pos).Error; err != nil {
		return nil, err
	}
	agents := make([]management.PostfixAgent, len(pos))
	for i := range pos {
		agents[i] = *poToPostfixAgent(&pos[i])
	}
	return agents, nil
}

func (r *PostfixAgentRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]management.PostfixAgent, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}
	db := g.Model(&PostfixAgentPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("name LIKE ? OR host LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := g.Model(&PostfixAgentPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("name LIKE ? OR host LIKE ?", like, like)
	}
	offset := (page - 1) * pageSize
	var pos []PostfixAgentPO
	if err := q.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "name"},
		Desc:   false,
	}).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	agents := make([]management.PostfixAgent, len(pos))
	for i := range pos {
		agents[i] = *poToPostfixAgent(&pos[i])
	}
	return agents, total, nil
}

func (r *PostfixAgentRepository) Delete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	result := g.Where("id = ?", id).Delete(&PostfixAgentPO{})
	if result.RowsAffected == 0 {
		return management.ErrPostfixAgentNotFound
	}
	return result.Error
}

func (r *PostfixAgentRepository) UpdateStatus(ctx context.Context, id shared.GlobalID, status string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Model(&PostfixAgentPO{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_status": status,
			"last_sync_at": time.Now(),
			"update_time":  time.Now(),
		}).Error
}

// --- PostfixConfig PO converters ---

func poToPostfixConfig(po *PostfixConfigPO) *management.PostfixConfig {
	if po == nil {
		return nil
	}
	return &management.PostfixConfig{
		ID:          po.ID,
		ParamName:   po.ParamName,
		ParamValue:  po.ParamValue,
		Category:    po.Category,
		IsManaged:   po.IsManaged,
		Enabled:     po.Enabled,
		Description: po.Description,
		SortOrder:   po.SortOrder,
		CreateTime:  po.CreateTime,
		UpdateTime:  po.UpdateTime,
	}
}

func postfixConfigToPO(c *management.PostfixConfig) *PostfixConfigPO {
	if c == nil {
		return nil
	}
	return &PostfixConfigPO{
		ID:          c.ID,
		ParamName:   c.ParamName,
		ParamValue:  c.ParamValue,
		Category:    c.Category,
		IsManaged:   c.IsManaged,
		Enabled:     c.Enabled,
		Description: c.Description,
		SortOrder:   c.SortOrder,
		CreateTime:  c.CreateTime,
		UpdateTime:  c.UpdateTime,
	}
}

// PostfixConfigRepository implements management.PostfixConfigRepository.
type PostfixConfigRepository struct {
	db persistence.DBProvider
}

func NewPostfixConfigRepository(db persistence.DBProvider) *PostfixConfigRepository {
	return &PostfixConfigRepository{db: db}
}

func (r *PostfixConfigRepository) Save(ctx context.Context, c *management.PostfixConfig) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	var exists int64
	if err := g.Model(&PostfixConfigPO{}).
		Where("param_name = ? AND id <> ?", c.ParamName, c.ID).
		Count(&exists).Error; err != nil {
		return err
	}
	if exists > 0 {
		return management.ErrPostfixConfigDuplicate
	}
	po := postfixConfigToPO(c)
	po.UpdateTime = time.Now()
	return g.Save(po).Error
}

func (r *PostfixConfigRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.PostfixConfig, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po PostfixConfigPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrPostfixConfigNotFound
		}
		return nil, err
	}
	return poToPostfixConfig(&po), nil
}

func (r *PostfixConfigRepository) FindByParamName(ctx context.Context, paramName string) (*management.PostfixConfig, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var po PostfixConfigPO
	if err := g.Where("param_name = ?", paramName).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, management.ErrPostfixConfigNotFound
		}
		return nil, err
	}
	return poToPostfixConfig(&po), nil
}

func (r *PostfixConfigRepository) FindAllManaged(ctx context.Context) ([]management.PostfixConfig, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []PostfixConfigPO
	if err := g.Where("is_managed = ? AND enabled = ?", true, true).
		Order("sort_order, param_name").Find(&pos).Error; err != nil {
		return nil, err
	}
	cfgs := make([]management.PostfixConfig, len(pos))
	for i := range pos {
		cfgs[i] = *poToPostfixConfig(&pos[i])
	}
	return cfgs, nil
}

func (r *PostfixConfigRepository) FindAllUserDefined(ctx context.Context) ([]management.PostfixConfig, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []PostfixConfigPO
	if err := g.Where("is_managed = ? AND enabled = ?", false, true).
		Order("sort_order, param_name").Find(&pos).Error; err != nil {
		return nil, err
	}
	cfgs := make([]management.PostfixConfig, len(pos))
	for i := range pos {
		cfgs[i] = *poToPostfixConfig(&pos[i])
	}
	return cfgs, nil
}

func (r *PostfixConfigRepository) FindAll(ctx context.Context) ([]management.PostfixConfig, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	var pos []PostfixConfigPO
	if err := g.Order("sort_order, param_name").Find(&pos).Error; err != nil {
		return nil, err
	}
	cfgs := make([]management.PostfixConfig, len(pos))
	for i := range pos {
		cfgs[i] = *poToPostfixConfig(&pos[i])
	}
	return cfgs, nil
}

func (r *PostfixConfigRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]management.PostfixConfig, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}
	db := g.Model(&PostfixConfigPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		db = db.Where("param_name LIKE ? OR description LIKE ?", like, like)
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q := g.Model(&PostfixConfigPO{}).Session(&gorm.Session{})
	if keyword != "" {
		like := "%" + keyword + "%"
		q = q.Where("param_name LIKE ? OR description LIKE ?", like, like)
	}
	offset := (page - 1) * pageSize
	var pos []PostfixConfigPO
	if err := q.Order(clause.OrderByColumn{
		Column: clause.Column{Name: "sort_order"},
		Desc:   false,
	}).Order(clause.OrderByColumn{
		Column: clause.Column{Name: "param_name"},
		Desc:   false,
	}).Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	cfgs := make([]management.PostfixConfig, len(pos))
	for i := range pos {
		cfgs[i] = *poToPostfixConfig(&pos[i])
	}
	return cfgs, total, nil
}

func (r *PostfixConfigRepository) Delete(ctx context.Context, id shared.GlobalID) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	// Prevent deletion of managed params
	var po PostfixConfigPO
	if err := g.Where("id = ?", id).First(&po).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return management.ErrPostfixConfigNotFound
		}
		return err
	}
	if po.IsManaged {
		return management.ErrPostfixConfigNotEditable
	}
	result := g.Where("id = ?", id).Delete(&PostfixConfigPO{})
	if result.RowsAffected == 0 {
		return management.ErrPostfixConfigNotFound
	}
	return result.Error
}

func (r *PostfixConfigRepository) DeleteByCategory(ctx context.Context, category string) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Where("category = ? AND is_managed = ?", category, false).
		Delete(&PostfixConfigPO{}).Error
}

// --- PostfixDeliveryLog PO converters ---

func poToPostfixDeliveryLog(po *PostfixDeliveryLogPO) *management.PostfixDeliveryLog {
	if po == nil {
		return nil
	}
	return &management.PostfixDeliveryLog{
		ID:             po.ID,
		AgentID:        po.AgentID,
		Action:         po.Action,
		Status:         po.Status,
		ConfigSnapshot: po.ConfigSnapshot,
		ErrorMessage:   po.ErrorMessage,
		CreatedAt:      po.CreatedAt,
	}
}

func postfixDeliveryLogToPO(l *management.PostfixDeliveryLog) *PostfixDeliveryLogPO {
	if l == nil {
		return nil
	}
	return &PostfixDeliveryLogPO{
		ID:             l.ID,
		AgentID:        l.AgentID,
		Action:         l.Action,
		Status:         l.Status,
		ConfigSnapshot: l.ConfigSnapshot,
		ErrorMessage:   l.ErrorMessage,
		CreatedAt:      l.CreatedAt,
	}
}

// PostfixDeliveryLogRepository implements management.PostfixDeliveryLogRepository.
type PostfixDeliveryLogRepository struct {
	db persistence.DBProvider
}

func NewPostfixDeliveryLogRepository(db persistence.DBProvider) *PostfixDeliveryLogRepository {
	return &PostfixDeliveryLogRepository{db: db}
}

func (r *PostfixDeliveryLogRepository) Save(ctx context.Context, l *management.PostfixDeliveryLog) error {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return err
	}
	return g.Save(postfixDeliveryLogToPO(l)).Error
}

func (r *PostfixDeliveryLogRepository) FindByAgent(ctx context.Context, agentID shared.GlobalID, limit int) ([]management.PostfixDeliveryLog, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 20
	}
	var pos []PostfixDeliveryLogPO
	if err := g.Where("agent_id = ?", agentID).
		Order("created_at DESC").
		Limit(limit).Find(&pos).Error; err != nil {
		return nil, err
	}
	logs := make([]management.PostfixDeliveryLog, len(pos))
	for i := range pos {
		logs[i] = *poToPostfixDeliveryLog(&pos[i])
		// Join agent name
		var agent PostfixAgentPO
		if err := g.Where("id = ?", pos[i].AgentID).First(&agent).Error; err == nil {
			logs[i].AgentName = agent.Name
		}
	}
	return logs, nil
}

func (r *PostfixDeliveryLogRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]management.PostfixDeliveryLog, int64, error) {
	g, err := GormDBFromProvider(ctx, r.db)
	if err != nil {
		return nil, 0, err
	}
	db := g.Model(&PostfixDeliveryLogPO{}).Session(&gorm.Session{})
	q := g.Model(&PostfixDeliveryLogPO{}).Session(&gorm.Session{})

	if keyword != "" {
		// Search by agent ID (for simplicity)
		db = db.Where("agent_id LIKE ?", "%"+keyword+"%")
		q = q.Where("agent_id LIKE ?", "%"+keyword+"%")
	}
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	var pos []PostfixDeliveryLogPO
	if err := q.Order("created_at DESC").
		Offset(offset).Limit(pageSize).Find(&pos).Error; err != nil {
		return nil, 0, err
	}
	logs := make([]management.PostfixDeliveryLog, len(pos))
	for i := range pos {
		logs[i] = *poToPostfixDeliveryLog(&pos[i])
	}
	return logs, total, nil
}