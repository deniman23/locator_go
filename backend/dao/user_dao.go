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

// GetAll возвращает список всех пользователей.
func (dao *UserDAO) GetAll() ([]models.User, error) {
	var users []models.User
	if err := dao.DB.Order("id ASC").Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}
