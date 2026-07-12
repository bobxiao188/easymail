package filter

import (
	"easymail/internal/domain/filter/classifier"
	"easymail/internal/infrastructure/filter/assets"
)

func classifyModelAssetsReady(m *classifier.Model) bool {
	var repo assets.Repository
	return repo.AssetsReady(*m)
}


