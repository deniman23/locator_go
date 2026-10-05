package dao

import (
	"locator/models"

	"gorm.io/gorm"
)

type UserDAO struct {
	DB *gorm.DB
}

// NewUserDAO создаёт новый экземпляр UserDAO.
func NewUserDAO(db *gorm.DB) *UserDAO {
	return &UserDAO{DB: db}
}

// Create вставляет нового пользователя в базу данных.
func (dao *UserDAO) Create(user *models.User) error {
	return dao.DB.Create(user).Error
}

func (dao *UserDAO) Update(user *models.User) error {
	return dao.DB.Save(user).Error
}

// GetByID возвращает пользователя по его ID.
func (dao *UserDAO) GetByID(id int) (*models.User, error) {
	var user models.User
	if err := dao.DB.First(&user, id).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByKeyLookup возвращает пользователя по SHA-256 ключа (один SELECT).
func (dao *UserDAO) GetByKeyLookup(lookup string) (*models.User, error) {
	var user models.User
	if err := dao.DB.Where("key_lookup = ?", lookup).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// ListMissingKeyLookup — только учётки, созданные до колонки key_lookup.
func (dao *UserDAO) ListMissingKeyLookup() ([]models.User, error) {
	var users []models.User
	if err := dao.DB.Where("key_lookup IS NULL OR key_lookup = ''").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// GetAll возвращает список всех пользователей.
func (dao *UserDAO) GetAll() ([]models.User, error) {
	var users []models.User
	if err := dao.DB.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
