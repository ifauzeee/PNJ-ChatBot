package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/pnj-anonymous-bot/internal/models"
)

func (d *DB) GetActiveRooms() ([]*models.Room, error) {
	rows, err := d.Query(`
		SELECT r.id, r.slug, r.name, r.description, r.is_active, r.created_at,
		       (SELECT COUNT(*) FROM room_members WHERE room_id = r.id) as member_count
		FROM rooms r
		WHERE r.is_active = TRUE
		ORDER BY member_count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rooms: %w", err)
	}
	defer rows.Close()

	var rooms []*models.Room
	for rows.Next() {
		r := &models.Room{}
		err := rows.Scan(&r.ID, &r.Slug, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt, &r.MemberCount)
		if err != nil {
			return nil, err
		}
		rooms = append(rooms, r)
	}
	return rooms, nil
}

func (d *DB) CreateRoom(slug, name, description string) (*models.Room, error) {
	_, err := d.Exec(
		`INSERT INTO rooms (slug, name, description, is_active, created_at) VALUES (?, ?, ?, TRUE, ?)`,
		slug, name, description, time.Now(),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create room: %w", err)
	}
	return d.GetRoomBySlug(slug)
}

func (d *DB) GetRoomBySlug(slug string) (*models.Room, error) {
	r := &models.Room{}
	err := d.QueryRow(`
		SELECT id, slug, name, description, is_active, created_at,
		       (SELECT COUNT(*) FROM room_members WHERE room_id = rooms.id) as member_count
		FROM rooms
		WHERE slug = ?
	`, slug).Scan(&r.ID, &r.Slug, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt, &r.MemberCount)

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
	err := d.QueryRow(`
		SELECT id, slug, name, description, is_active, created_at,
		       (SELECT COUNT(*) FROM room_members WHERE room_id = rooms.id) as member_count
		FROM rooms
		WHERE id = ?
	`, id).Scan(&r.ID, &r.Slug, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt, &r.MemberCount)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}

func (d *DB) AddRoomMember(roomID int64, telegramID int64) error {
	_, err := d.Exec(
		`INSERT OR IGNORE INTO room_members (room_id, telegram_id, joined_at) VALUES (?, ?, ?)`,
		roomID, telegramID, time.Now(),
	)
	return err
}

func (d *DB) RemoveRoomMember(roomID int64, telegramID int64) error {
	_, err := d.Exec(`DELETE FROM room_members WHERE room_id = ? AND telegram_id = ?`, roomID, telegramID)
	return err
}

func (d *DB) RemoveMemberFromAllRooms(telegramID int64) error {
	_, err := d.Exec(`DELETE FROM room_members WHERE telegram_id = ?`, telegramID)
	return err
}

func (d *DB) GetRoomMembers(roomID int64) ([]int64, error) {
	rows, err := d.Query(`SELECT telegram_id FROM room_members WHERE room_id = ?`, roomID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func (d *DB) GetUserRoom(telegramID int64) (*models.Room, error) {
	r := &models.Room{}
	err := d.QueryRow(`
		SELECT r.id, r.slug, r.name, r.description, r.is_active, r.created_at
		FROM rooms r
		JOIN room_members rm ON r.id = rm.room_id
		WHERE rm.telegram_id = ?
	`, telegramID).Scan(&r.ID, &r.Slug, &r.Name, &r.Description, &r.IsActive, &r.CreatedAt)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return r, nil
}
