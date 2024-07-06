package auth

import (
	"errors"
	"os"
	"path"

	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

var (
	ErrNotFound = errors.New("User not found")
)

type Users interface {
	AddUser(username, password string) error
	CheckPassword(username, password string) bool
}

func NewUsers(dataDir string) (Users, error) {
	u := &users{
		filename: path.Join(dataDir, "users.yml"),
		store:    make([]*User, 0),
		dict:     make(map[string]*User),
	}
	err := u.Read()
	if err != nil {
		return nil, err
	}
	return u, nil
}

type User struct {
	Name     string
	Password string
	Mail     string
	Groups   []string
}

type users struct {
	filename string
	store    []*User
	dict     map[string]*User
}

func (u *users) Write() (err error) {
	data, err := yaml.Marshal(u.store)
	if err != nil {
		return err
	}
	return os.WriteFile(u.filename, data, 0o644)
}

func (u *users) Read() (err error) {
	data, err := os.ReadFile(u.filename)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	err = yaml.Unmarshal(data, &u.store)
	if err != nil {
		return err
	}
	for _, user := range u.store {
		u.dict[user.Name] = user
	}
	return nil
}

func (u *users) AddUser(username, password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost-2, // DefaultCost takes to long on a Raspberry-Pi
	)
	if err != nil {
		return err
	}

	user := u.dict[username]
	if user == nil {
		user = &User{Name: username}
		u.dict[username] = user
		u.store = append(u.store, user)
	}
	user.Password = string(hash)

	return u.Write()
}

func (u *users) CheckPassword(username, password string) bool {
	user := u.dict[username]
	if nil == user {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}
