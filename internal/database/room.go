package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) GetActiveRooms(ctx context.Context) ([]*models.Room, error) {
	subQuery := d.Builder.Select("COUNT(*)").From("room_members").Where("room_id = r.id")
	q, _, _ := subQuery.ToSql()

	builder := d.Builder.Select("r.id", "r.slug", "r.name", "r.description", "r.is_active", "r.created_at",
		"("+q+") as member_count").
		From("rooms r").
		Where("r.is_active = TRUE").
		OrderBy("member_count DESC")

	var rooms []*models.Room
	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}
	err = d.SelectContext(ctx, &rooms, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rooms: %w", err)
	}

	return rooms, nil
}

func (d *DB) CreateRoom(ctx context.Context, slug, name, description string) (*models.Room, error) {
	builder := d.Builder.Insert("rooms").
		Columns("slug", "name", "description", "is_active", "created_at").
		Values(slug, name, description, true, time.Now())

	_, err := d.ExecBuilderContext(ctx, builder)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	return d.GetRoomBySlug(ctx, slug)
}

func (d *DB) GetRoomBySlug(ctx context.Context, slug string) (*models.Room, error) {
	r := &models.Room{}
	subQuery := d.Builder.Select("COUNT(*)").From("room_members").Where("room_id = rooms.id")
	q, _, _ := subQuery.ToSql()

	builder := d.Builder.Select("id", "slug", "name", "description", "is_active", "created_at",
		"("+q+") as member_count").
		From("rooms").
		Where("slug = ?", slug)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	err = d.GetContext(ctx, r, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *DB) GetRoomByID(ctx context.Context, id int64) (*models.Room, error) {
	r := &models.Room{}
	subQuery := d.Builder.Select("COUNT(*)").From("room_members").Where("room_id = rooms.id")
	q, _, _ := subQuery.ToSql()

	builder := d.Builder.Select("id", "slug", "name", "description", "is_active", "created_at",
		"("+q+") as member_count").
		From("rooms").
		Where("id = ?", id)

	query, args, err := builder.ToSql()
	if err != nil {
		return nil, err
	}

	err = d.GetContext(ctx, r, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *DB) AddRoomMember(ctx context.Context, roomID int64, telegramID int64) error {
	builder := d.Builder.Insert("room_members").
		Columns("room_id", "telegram_id", "joined_at").
		Values(roomID, telegramID, time.Now())

	_, err := d.InsertIgnoreContext(ctx, builder, "room_id, telegram_id")
	return err
}

func (d *DB) RemoveRoomMember(ctx context.Context, roomID int64, telegramID int64) error {
	builder := d.Builder.Delete("room_members").Where("room_id = ? AND telegram_id = ?", roomID, telegramID)
	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) RemoveMemberFromAllRooms(ctx context.Context, telegramID int64) error {
	builder := d.Builder.Delete("room_members").Where("telegram_id = ?", telegramID)
	_, err := d.ExecBuilderContext(ctx, builder)
	return err
}

func (d *DB) GetRoomMembers(ctx context.Context, roomID int64) ([]int64, error) {
	var ids []int64
	builder := d.Builder.Select("telegram_id").From("room_members").Where("room_id = ?", roomID)
	err := d.SelectBuilderContext(ctx, &ids, builder)
	return ids, err
}

func (d *DB) GetUserRoom(ctx context.Context, telegramID int64) (*models.Room, error) {
	r := &models.Room{}
	builder := d.Builder.Select("r.id", "r.slug", "r.name", "r.description", "r.is_active", "r.created_at").
		From("rooms r").
		Join("room_members rm ON r.id = rm.room_id").
		Where("rm.telegram_id = ?", telegramID)

	err := d.GetBuilderContext(ctx, r, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}
