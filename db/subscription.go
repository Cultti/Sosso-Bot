package db

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

type Subscription struct {
	ID        uuid.UUID `gorm:"type:char(36);primaryKey"`
	GuildID   string    `gorm:"size:64;not null"`
	ChannelID string    `gorm:"size:64;not null"`
	League    string    `gorm:"size:50;not null"`
	CreatedAt time.Time // Automatically managed by GORM for creation time
	UpdatedAt time.Time // Automatically managed by GORM for update time
}

func CreateSubscription(s *Subscription) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return database.Create(s).Error
}

func GetSubscription(id uuid.UUID) (*Subscription, error) {
	var s Subscription
	if err := database.First(&s, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &s, nil
}

func UpdateSubscription(id uuid.UUID, newSub Subscription) error {
	var s Subscription
	if err := database.First(&s, "id = ?", id).Error; err != nil {
		return err
	}
	newSub.ID = s.ID
	return database.Save(&newSub).Error
}

func DeleteSubscription(id uuid.UUID) error {
	return database.Delete(&Subscription{}, "id = ?", id).Error
}

func GetSubscriptionsByLeague(league string) (*[]Subscription, error) {
	var subs []Subscription

	// Handle cases where the incoming league contains "Playoffs "
	// We want to match subscriptions that could be stored as either:
	// - "20 Divisioona S11" (normalized, without "Playoffs ")
	// - "20 Divisioona Playoffs S11" (exact match)
	normalizedLeague := strings.Replace(league, "Playoffs ", "", 1)

	// Search for subscriptions that match either the exact league name or the normalized version
	err := database.Where("league = ? OR league = ?", league, normalizedLeague).Find(&subs).Error
	return &subs, err
}

func DeleteSubscriptionsByGuildChannel(guildID, channelID, league string) ([]string, error) {
	var subs []Subscription

	// Build query
	q := database.Where("guild_id = ? AND channel_id = ?", guildID, channelID)
	if league != "" {
		q = q.Where("league = ?", league)
	}

	// Fetch subscriptions to know which leagues are being deleted
	if err := q.Find(&subs).Error; err != nil {
		return nil, err
	}

	if len(subs) == 0 {
		return nil, nil // nothing to delete
	}

	// Collect league names
	deletedLeagues := make([]string, 0, len(subs))
	for _, s := range subs {
		deletedLeagues = append(deletedLeagues, s.League)
	}

	// Delete subscriptions
	if err := q.Delete(&Subscription{}).Error; err != nil {
		return nil, err
	}

	return deletedLeagues, nil
}

func GetSubscriptionsByGuildChannel(guildID string, channelID string) (*[]Subscription, error) {
	var subs []Subscription

	// Build query
	q := database.Where("guild_id = ? AND channel_id = ?", guildID, channelID)

	if err := q.Find(&subs).Error; err != nil {
		return nil, err
	}

	return &subs, nil
}
