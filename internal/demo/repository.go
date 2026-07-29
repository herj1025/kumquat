package demo

import (
	"context"

	"github.com/herj1025/kumquat/pkg/e"

	"gorm.io/gorm"
)

type repository struct {
	db *gorm.DB
}

func newRepository(gormDB *gorm.DB) *repository {
	return &repository{
		db: gormDB,
	}
}

func (r *repository) FindById(ctx context.Context) (*Demo, error) {
	demo, err := gorm.G[Demo](r.db).Where(
		"id = ?", 1,
	).Take(ctx)
	if err != nil {
		return nil, e.New(5000, "查询数据错误")
	}
	return &demo, nil
}
