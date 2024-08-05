package model

import (
	"log"
	"time"

	database "remote-diffusion-server/database"
	utils "remote-diffusion-server/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RegisteredKeyModel struct {
	db        *gorm.DB
	ID        int       `gorm:"primaryKey"`
	UserKey   uuid.UUID `gorm:"type:uuid"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
}

func (RegisteredKeyModel) TableName() string {
	return "registered_keys"
}

func NewRegisteredKeys() RegisteredKeyModel {
	db, err := database.ConnectToPostgres()
	if err != nil {
		log.Fatal("Error connecting to the database:", err)
	}
	//db.AutoMigrate(&RegisteredKeyModel{})
	return RegisteredKeyModel{
		db: db,
	}
}

func (keys *RegisteredKeyModel) Get(key string) time.Time {
	if !utils.IsValidUUID(key) {
		log.Println("Invalid user key: ", key)
		return time.Time{}
	}

	var registeredKey RegisteredKeyModel
	err := keys.db.Where("user_key = ?", key).First(&registeredKey).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No rows were returned - return a zero time.Time value
			return time.Time{}
		}
		log.Println("Error getting user key from the database:", err)
		return time.Time{}
	}

	return registeredKey.CreatedAt
}

func (keys *RegisteredKeyModel) Set(key string, value time.Time) {
	if !utils.IsValidUUID(key) {
		log.Println("Invalid user key: ", key)
	}

	registeredKey := RegisteredKeyModel{
		UserKey:   uuid.MustParse(key),
		CreatedAt: value,
		UpdatedAt: value,
	}
	err := keys.db.Create(&registeredKey).Error
	if err != nil {
		log.Println("Error inserting user key into the database:", err)
	}
}

func (keys *RegisteredKeyModel) Remove(key string) {
	if !utils.IsValidUUID(key) {
		log.Println("Invalid user key: ", key)
	}

	err := keys.db.Where("user_key = ?", key).Delete(&RegisteredKeyModel{}).Error
	if err != nil {
		log.Println("Error deleting user key from the database:", err)
	}
}

func (keys *RegisteredKeyModel) Count() (int64, error) {
	var count int64
	err := keys.db.Model(&RegisteredKeyModel{}).Count(&count).Error
	if err != nil {
		log.Println("Error getting count from the database:", err)
		return 0, err
	}

	return count, nil
}
