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

func mailUserCacheKey(scope, key string) string {
	return "mailuser:" + scope + ":" + key
}

// CachedMailUserRepository wraps a management.MailUserRepository with a CacheBackend.
type CachedMailUserRepository struct {
	repo  management.MailUserRepository
	cache CacheBackend
	ttl   time.Duration
	sf    singleflight.Group
}

func NewCachedMailUserRepository(repo management.MailUserRepository, cache CacheBackend, ttl time.Duration) *CachedMailUserRepository {
	return &CachedMailUserRepository{repo: repo, cache: cache, ttl: ttl}
}

func (r *CachedMailUserRepository) isEnabled() bool {
	return r.ttl > 0 && r.cache != nil
}

// ---- Cache-aside reads ----

func (r *CachedMailUserRepository) FindByID(ctx context.Context, id shared.GlobalID) (*management.MailUser, error) {
	if !r.isEnabled() {
		return r.repo.FindByID(ctx, id)
	}
	key := mailUserCacheKey("id", id.String())
	return cachedFind(ctx, r.cache, &r.sf, key, r.ttl, func() (*management.MailUser, error) {
		return r.repo.FindByID(ctx, id)
	})
}

func (r *CachedMailUserRepository) FindByFullEmail(ctx context.Context, email string) (*management.MailUser, error) {
	if !r.isEnabled() {
		return r.repo.FindByFullEmail(ctx, email)
	}
	key := mailUserCacheKey("email", email)
	return cachedFind(ctx, r.cache, &r.sf, key, r.ttl, func() (*management.MailUser, error) {
		return r.repo.FindByFullEmail(ctx, email)
	})
}

func (r *CachedMailUserRepository) FindByUsername(ctx context.Context, domainID shared.GlobalID, username string) (*management.MailUser, error) {
	if !r.isEnabled() {
		return r.repo.FindByUsername(ctx, domainID, username)
	}
	key := mailUserCacheKey("username", domainID.String()+":"+username)
	return cachedFind(ctx, r.cache, &r.sf, key, r.ttl, func() (*management.MailUser, error) {
		return r.repo.FindByUsername(ctx, domainID, username)
	})
}

// ---- Passthrough ----

func (r *CachedMailUserRepository) Search(ctx context.Context, domainID shared.GlobalID, keyword string, status int, page, pageSize int) ([]management.MailUser, int64, error) {
	return r.repo.Search(ctx, domainID, keyword, status, page, pageSize)
}

func (r *CachedMailUserRepository) FindByDomainID(ctx context.Context, domainID shared.GlobalID) ([]management.MailUser, error) {
	return r.repo.FindByDomainID(ctx, domainID)
}

// ---- Write-through + invalidation ----

func (r *CachedMailUserRepository) Save(ctx context.Context, u *management.MailUser) error {
	oldEmail, _ := r.loadOldEmail(ctx, u.ID)
	if err := r.repo.Save(ctx, u); err != nil {
		return err
	}
	r.invalidateUser(ctx, u.ID, oldEmail, u.Email)
	return nil
}

func (r *CachedMailUserRepository) SoftDelete(ctx context.Context, id shared.GlobalID) error {
	oldEmail, _ := r.loadOldEmail(ctx, id)
	if err := r.repo.SoftDelete(ctx, id); err != nil {
		return err
	}
	r.invalidateUser(ctx, id, oldEmail, "")
	return nil
}

func (r *CachedMailUserRepository) HardDelete(ctx context.Context, id shared.GlobalID) error {
	oldEmail, _ := r.loadOldEmail(ctx, id)
	if err := r.repo.HardDelete(ctx, id); err != nil {
		return err
	}
	r.invalidateUser(ctx, id, oldEmail, "")
	return nil
}

func (r *CachedMailUserRepository) ToggleActive(ctx context.Context, id shared.GlobalID) error {
	oldEmail, _ := r.loadOldEmail(ctx, id)
	if err := r.repo.ToggleActive(ctx, id); err != nil {
		return err
	}
	r.invalidateUser(ctx, id, oldEmail, "")
	return nil
}

func (r *CachedMailUserRepository) UpdatePassword(ctx context.Context, id shared.GlobalID, hash string) error {
	oldEmail, _ := r.loadOldEmail(ctx, id)
	if err := r.repo.UpdatePassword(ctx, id, hash); err != nil {
		return err
	}
	r.invalidateUser(ctx, id, oldEmail, "")
	return nil
}

func (r *CachedMailUserRepository) ChangePassword(ctx context.Context, id shared.GlobalID, oldPassword, newPassword string) error {
	oldEmail, _ := r.loadOldEmail(ctx, id)
	if err := r.repo.ChangePassword(ctx, id, oldPassword, newPassword); err != nil {
		return err
	}
	r.invalidateUser(ctx, id, oldEmail, "")
	return nil
}

func (r *CachedMailUserRepository) loadOldEmail(ctx context.Context, id shared.GlobalID) (string, bool) {
	if !r.isEnabled() {
		return "", false
	}
	key := mailUserCacheKey("id", id.String())
	if d := getUserFromCache(ctx, r.cache, key); d != nil {
		return d.Email, true
	}
	d, err := r.repo.FindByID(ctx, id)
	if err != nil {
		return "", false
	}
	return d.Email, true
}

func (r *CachedMailUserRepository) invalidateUser(ctx context.Context, id shared.GlobalID, oldEmail, newEmail string) {
	keys := []string{mailUserCacheKey("id", id.String())}
	if oldEmail != "" {
		keys = append(keys, mailUserCacheKey("email", oldEmail))
	}
	if newEmail != "" && newEmail != oldEmail {
		keys = append(keys, mailUserCacheKey("email", newEmail))
	}
	_ = r.cache.Del(ctx, keys...)
}

// ---- JSON serialization ----

type jsonMailUser struct {
	ID               string `json:"id"`
	DomainID         string `json:"domainId"`
	Username         string `json:"username"`
	PasswordHash     string `json:"passwordHash"`
	Email            string `json:"email"`
	Active           bool   `json:"active"`
	IsDeleted        bool   `json:"isDeleted"`
	StorageQuota     int64  `json:"storageQuota"`
	DataPath         string `json:"dataPath"`
	StorageID        int    `json:"storageId"`
	PasswordExpireAt string `json:"passwordExpireAt"`
	CreateTime       string `json:"createTime"`
	UpdateTime       string `json:"updateTime"`
	DeleteTime       string `json:"deleteTime"`
}

func mailUserToJSON(u *management.MailUser) jsonMailUser {
	return jsonMailUser{
		ID:               u.ID.String(),
		DomainID:         u.DomainID.String(),
		Username:         u.Username,
		PasswordHash:     u.PasswordHash,
		Email:            u.Email,
		Active:           u.Active,
		IsDeleted:        u.IsDeleted,
		StorageQuota:     u.StorageQuota,
		DataPath:         u.DataPath,
		StorageID:        u.StorageID,
		PasswordExpireAt: timeutil.FormatJSON(u.PasswordExpireAt),
		CreateTime:       timeutil.FormatJSON(u.CreateTime),
		UpdateTime:       timeutil.FormatJSON(u.UpdateTime),
		DeleteTime:       timeutil.FormatJSON(u.DeleteTime),
	}
}

func mailUserFromJSON(j jsonMailUser) *management.MailUser {
	id, _ := shared.ParseGlobalID(j.ID)
	domainID, _ := shared.ParseGlobalID(j.DomainID)
	pe, _ := timeutil.ParseJSON(j.PasswordExpireAt)
	ct, _ := timeutil.ParseJSON(j.CreateTime)
	ut, _ := timeutil.ParseJSON(j.UpdateTime)
	dt, _ := timeutil.ParseJSON(j.DeleteTime)
	return &management.MailUser{
		ID:               id,
		DomainID:         domainID,
		Username:         j.Username,
		PasswordHash:     j.PasswordHash,
		Email:            j.Email,
		Active:           j.Active,
		IsDeleted:        j.IsDeleted,
		StorageQuota:     j.StorageQuota,
		DataPath:         j.DataPath,
		StorageID:        j.StorageID,
		PasswordExpireAt: pe,
		CreateTime:       ct,
		UpdateTime:       ut,
		DeleteTime:       dt,
	}
}

func getUserFromCache(ctx context.Context, cache CacheBackend, key string) *management.MailUser {
	data, found, err := cache.Get(ctx, key)
	if err != nil || !found {
		return nil
	}
	var j jsonMailUser
	if json.Unmarshal(data, &j) != nil {
		return nil
	}
	return mailUserFromJSON(j)
}

func setUserToCache(ctx context.Context, cache CacheBackend, key string, u *management.MailUser, ttl time.Duration) {
	data, err := json.Marshal(mailUserToJSON(u))
	if err != nil {
		return
	}
	_ = cache.Set(ctx, key, data, ttl)
}

func cachedFind(ctx context.Context, cache CacheBackend, sf *singleflight.Group, key string, ttl time.Duration, fn func() (*management.MailUser, error)) (*management.MailUser, error) {
	if d := getUserFromCache(ctx, cache, key); d != nil {
		return d, nil
	}
	v, err, _ := sf.Do(key, func() (interface{}, error) {
		if d := getUserFromCache(ctx, cache, key); d != nil {
			return d, nil
		}
		return fn()
	})
	if err != nil {
		return nil, err
	}
	d := v.(*management.MailUser)
	setUserToCache(ctx, cache, key, d, ttl)
	return d, nil
}

var _ management.MailUserRepository = (*CachedMailUserRepository)(nil)
