package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) GetActiveRooms() ([]*models.Room, error) {
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
	err = d.Select(&rooms, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rooms: %w", err)
	}

	return rooms, nil
}

func (d *DB) CreateRoom(slug, name, description string) (*models.Room, error) {
	builder := d.Builder.Insert("rooms").
		Columns("slug", "name", "description", "is_active", "created_at").
		Values(slug, name, description, true, time.Now())

	_, err := d.ExecBuilder(builder)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	return d.GetRoomBySlug(slug)
}

func (d *DB) GetRoomBySlug(slug string) (*models.Room, error) {
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

	err = d.Get(r, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *DB) GetRoomByID(id int64) (*models.Room, error) {
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

	err = d.Get(r, query, args...)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *DB) AddRoomMember(roomID int64, telegramID int64) error {
	builder := d.Builder.Insert("room_members").
		Columns("room_id", "telegram_id", "joined_at").
		Values(roomID, telegramID, time.Now())

	_, err := d.InsertIgnore(builder, "room_id, telegram_id")
	return err
}

func (d *DB) RemoveRoomMember(roomID int64, telegramID int64) error {
	builder := d.Builder.Delete("room_members").Where("room_id = ? AND telegram_id = ?", roomID, telegramID)
	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) RemoveMemberFromAllRooms(telegramID int64) error {
	builder := d.Builder.Delete("room_members").Where("telegram_id = ?", telegramID)
	_, err := d.ExecBuilder(builder)
	return err
}

func (d *DB) GetRoomMembers(roomID int64) ([]int64, error) {
	var ids []int64
	builder := d.Builder.Select("telegram_id").From("room_members").Where("room_id = ?", roomID)
	err := d.SelectBuilder(&ids, builder)
	return ids, err
}

func (d *DB) GetUserRoom(telegramID int64) (*models.Room, error) {
	r := &models.Room{}
	builder := d.Builder.Select("r.id", "r.slug", "r.name", "r.description", "r.is_active", "r.created_at").
		From("rooms r").
		Join("room_members rm ON r.id = rm.room_id").
		Where("rm.telegram_id = ?", telegramID)

	err := d.GetBuilder(r, builder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}
