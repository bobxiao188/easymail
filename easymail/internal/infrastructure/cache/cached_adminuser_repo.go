package cache

import (
	"context"
	"encoding/json"
	"time"

	"easymail/internal/domain/management"
	"easymail/internal/domain/shared"
	"easymail/pkg/timeutil"

	"golang.org/x/sync/singleflight"
)

func adminCacheKey(scope, key string) string {
	return "admin:" + scope + ":" + key
}

// CachedAdminUserRepository wraps a management.AdminUserRepository with a CacheBackend.
type CachedAdminUserRepository struct {
	repo  management.AdminUserRepository
	cache CacheBackend
	ttl   time.Duration
	sf    singleflight.Group
}

func NewCachedAdminUserRepository(repo management.AdminUserRepository, cache CacheBackend, ttl time.Duration) *CachedAdminUserRepository {
	return &CachedAdminUserRepository{repo: repo, cache: cache, ttl: ttl}
}

func (r *CachedAdminUserRepository) isEnabled() bool {
	return r.ttl > 0 && r.cache != nil
}

// ---- Cache-aside reads ----

func (r *CachedAdminUserRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.AdminUser, error) {
	if !r.isEnabled() {
		return r.repo.FindByID(ctx, id)
	}
	key := adminCacheKey("id", id.String())
	return cachedFindAdmin(ctx, r.cache, &r.sf, key, r.ttl, func() (*management.AdminUser, error) {
		return r.repo.FindByID(ctx, id)
	})
}

func (r *CachedAdminUserRepository) FindByUsername(ctx context.Context, username string) (*management.AdminUser, error) {
	if !r.isEnabled() {
		return r.repo.FindByUsername(ctx, username)
	}
	key := adminCacheKey("username", username)
	return cachedFindAdmin(ctx, r.cache, &r.sf, key, r.ttl, func() (*management.AdminUser, error) {
		return r.repo.FindByUsername(ctx, username)
	})
}

// ---- Passthrough ----

func (r *CachedAdminUserRepository) Search(ctx context.Context, keyword string, page, pageSize int) ([]management.AdminUser, int64, error) {
	return r.repo.Search(ctx, keyword, page, pageSize)
}

// ---- Write-through + invalidation ----

func (r *CachedAdminUserRepository) Save(ctx context.Context, u *management.AdminUser) error {
	oldUsername, _ := r.loadOldUsername(ctx, u.ID)
	if err := r.repo.Save(ctx, u); err != nil {
		return err
	}
	r.invalidateAdmin(ctx, u.ID, oldUsername, u.Username)
	return nil
}

func (r *CachedAdminUserRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	oldUsername, _ := r.loadOldUsername(ctx, id)
	if err := r.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	r.invalidateAdmin(ctx, id, oldUsername, "")
	return nil
}

func (r *CachedAdminUserRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	oldUsername, _ := r.loadOldUsername(ctx, id)
	if err := r.repo.ToggleActive(ctx, id); err != nil {
		return err
	}
	r.invalidateAdmin(ctx, id, oldUsername, "")
	return nil
}

func (r *CachedAdminUserRepository) UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error {
	oldUsername, _ := r.loadOldUsername(ctx, id)
	if err := r.repo.UpdatePassword(ctx, id, hash); err != nil {
		return err
	}
	r.invalidateAdmin(ctx, id, oldUsername, "")
	return nil
}

func (r *CachedAdminUserRepository) loadOldUsername(ctx context.Context, id shared.GlobalID) (string, bool) {
	if !r.isEnabled() {
		return "", false
	}
	key := adminCacheKey("id", id.String())
	if d := getAdminFromCache(ctx, r.cache, key); d != nil {
		return d.Username, true
	}
	d, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return "", false
	}
	return d.Username, true
}

func (r *CachedAdminUserRepository) invalidateAdmin(ctx context.Context, id shared.GlobalID, oldUsername, newUsername string) {
	keys := []string{adminCacheKey("id", id.String())}
	if oldUsername != "" {
		keys = append(keys, adminCacheKey("username", oldUsername))
	}
	if newUsername != "" && newUsername != oldUsername {
		keys = append(keys, adminCacheKey("username", newUsername))
	}
	_ = r.cache.Del(ctx, keys...)
}

// ---- JSON serialization ----

type jsonAdminUser struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	PasswordHash string `json:"passwordHash"`
	Nickname     string `json:"nickname"`
	Email        string `json:"email"`
	Avatar       string `json:"avatar"`
	Language     string `json:"language"`
	Skin         string `json:"skin"`
	Active       bool   `json:"active"`
	IsDeleted    bool   `json:"isDeleted"`
	CreateTime   string `json:"createTime"`
	UpdateTime   string `json:"updateTime"`
	DeleteTime   string `json:"deleteTime"`
}

func adminUserToJSON(u *management.AdminUser) jsonAdminUser {
	return jsonAdminUser{
		ID:           u.ID.String(),
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		Nickname:     u.Nickname,
		Email:        u.Email,
		Avatar:       u.Avatar,
		Language:     u.Language,
		Skin:         u.Skin,
		Active:       u.Active,
		IsDeleted:    u.IsDeleted,
		CreateTime:   timeutil.FormatJSON(u.CreateTime),
		UpdateTime:   timeutil.FormatJSON(u.UpdateTime),
		DeleteTime:   timeutil.FormatJSON(u.DeleteTime),
	}
}

func adminUserFromJSON(j jsonAdminUser) *management.AdminUser {
	id, _ := shared.ParseGlobalID(j.ID)
	ct, _ := timeutil.ParseJSON(j.CreateTime)
	ut, _ := timeutil.ParseJSON(j.UpdateTime)
	dt, _ := timeutil.ParseJSON(j.DeleteTime)
	return &management.AdminUser{
		ID:           id,
		Username:     j.Username,
		PasswordHash: j.PasswordHash,
		Nickname:     j.Nickname,
		Email:        j.Email,
		Avatar:       j.Avatar,
		Language:     j.Language,
		Skin:         j.Skin,
		Active:       j.Active,
		IsDeleted:    j.IsDeleted,
		CreateTime:   ct,
		UpdateTime:   ut,
		DeleteTime:   dt,
	}
}

func getAdminFromCache(ctx context.Context, cache CacheBackend, key string) *management.AdminUser {
	data, found, err := cache.Get(ctx, key)
	if err != nil || !found {
		return nil
	}
	var j jsonAdminUser
	if json.Unmarshal(data, &j) != nil {
		return nil
	}
	return adminUserFromJSON(j)
}

func setAdminToCache(ctx context.Context, cache CacheBackend, key string, u *management.AdminUser, ttl time.Duration) {
	data, err := json.Marshal(adminUserToJSON(u))
	if err != nil {
		return
	}
	_ = cache.Set(ctx, key, data, ttl)
}

func cachedFindAdmin(ctx context.Context, cache CacheBackend, sf *singleflight.Group, key string, ttl time.Duration, fn func() (*management.AdminUser, error)) (*management.AdminUser, error) {
	if d := getAdminFromCache(ctx, cache, key); d != nil {
		return d, nil
	}
	v, err, _ := sf.Do(key, func() (interface{}, error) {
		if d := getAdminFromCache(ctx, cache, key); d != nil {
			return d, nil
		}
		return fn()
	})
	if err != nil {
		return nil, err
	}
	d := v.(*management.AdminUser)
	setAdminToCache(ctx, cache, key, d, ttl)
	return d, nil
}

var _ management.AdminUserRepository = (*CachedAdminUserRepository)(nil)

