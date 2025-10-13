package auth

import (
	"crypto/rand"
	"encoding/base32"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gopkg.in/yaml.v3"
)

var (
	ErrNotFound = errors.New("User not found")
)

type Users interface {
	Users() []User
	AddUser(user User) (User, error)
	DeleteUser(guid string) error
	CheckPassword(username, password string) bool

	StartPasswordReset(mail string) (user User, err error)
	PasswordReset(token string, password string) (err error)

	GetUser(guid string) (*User, error)

	Groups() []Group
	AddGroup(name string) (Group, error)
	DeleteGroup(guid string) error
}

func NewUsers(dataDir string) (Users, error) {
	u := &users{
		filename:          path.Join(dataDir, "users.yml"),
		usersByGuid:       make(map[string]*User),
		usersByNameOrMail: make(map[string]*User),
		groupsByGuid:      make(map[string]*Group),
		groupsByName:      make(map[string]*Group),
	}
	err := u.Read()
	if err != nil {
		return nil, err
	}
	return u, nil
}

type User struct {
	Guid          string     `yaml:"guid" json:"guid"`
	Name          string     `yaml:"name" json:"name"`
	Password      string     `yaml:"password,omitempty" json:"password,omitempty"`
	Mail          string     `yaml:"mail" json:"mail"`
	Groups        []string   `yaml:"groups" json:"groups"`
	ResetToken    string     `yaml:"reset_token,omitempty" json:"reset_token,omitempty"`
	ResetTokenTTL *time.Time `yaml:"reset_token_ttl,omitempty" json:"reset_token_ttl,omitempty"`
}

func (user *User) SetPassword(password string) error {
	hash, err := bcrypt.GenerateFromPassword([]byte(password),
		bcrypt.DefaultCost-2, // DefaultCost takes to long on a Raspberry-Pi
	)
	if err != nil {
		return err
	}
	user.Password = string(hash)
	return nil
}

type Group struct {
	Guid string `yaml:"guid" json:"guid"`
	Name string `yaml:"name" json:"name"`
}

type Config struct {
	Users  []*User  `yaml:"users" json:"users"`
	Groups []*Group `yaml:"groups" json:"groups"`
}

type users struct {
	filename string
	config   Config

	usersByGuid       map[string]*User
	usersByNameOrMail map[string]*User

	groupsByGuid map[string]*Group
	groupsByName map[string]*Group
}

func (u *users) Write() (err error) {
	data, err := yaml.Marshal(u.config)
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
	err = yaml.Unmarshal(data, &u.config)
	if err != nil {
		return err
	}

	u.usersByGuid = make(map[string]*User)
	u.usersByNameOrMail = make(map[string]*User)
	u.groupsByGuid = make(map[string]*Group)
	u.groupsByName = make(map[string]*Group)

	mustSave := false
	for _, user := range u.config.Users {
		if user.Guid == "" {
			mustSave = true
			user.Guid = uuid.NewString()
		}
		u.usersByGuid[user.Guid] = user
		if user.Name != "" {
			u.usersByNameOrMail[strings.ToLower(user.Name)] = user
		}
		if user.Mail != "" {
			u.usersByNameOrMail[strings.ToLower(user.Mail)] = user
		}
	}

	for _, group := range u.config.Groups {
		if group.Guid == "" {
			mustSave = true
			group.Guid = uuid.NewString()
		}
		u.groupsByGuid[group.Guid] = group
		if group.Name != "" {
			u.groupsByName[group.Name] = group
		}
	}

	if mustSave {
		return u.Write()
	}

	return nil
}

func (u *users) Users() []User {
	users := make([]User, 0, len(u.config.Users))
	for _, user := range u.config.Users {
		userWithoutPassword := *user
		userWithoutPassword.Password = ""
		users = append(users, userWithoutPassword)
	}
	return users
}

func (u *users) AddUser(user User) (User, error) {
	if user.Name == "" {
		return User{}, fmt.Errorf("Users must have a name")
	}
	if user.Mail == "" {
		return User{}, fmt.Errorf("Users must have an email")
	}
	if len(user.Groups) == 0 {
		user.Groups = []string{"user"}
	}
	err := u.CheckGroups(user.Groups...)
	if err != nil {
		return User{}, err
	}

	existingUser := u.usersByNameOrMail[strings.ToLower(user.Name)]
	if existingUser == nil {
		existingUser = u.usersByNameOrMail[strings.ToLower(user.Mail)]
	}
	if existingUser != nil {
		return User{}, fmt.Errorf("User already exists")
	}

	if user.Password != "" {
		err = user.SetPassword(user.Password)
		if err != nil {
			return User{}, err
		}
	}

	user.Guid = uuid.New().String()

	u.usersByNameOrMail[strings.ToLower(user.Name)] = &user
	u.usersByNameOrMail[strings.ToLower(user.Mail)] = &user

	u.config.Users = append(u.config.Users, &user)

	err = u.Write()
	if err != nil {
		return User{}, err
	}

	userWithoutPassword := user
	userWithoutPassword.Password = ""

	return userWithoutPassword, nil
}

func (u *users) DeleteUser(guid string) error {
	user := u.usersByGuid[guid]
	if user == nil {
		return ErrNotFound
	}
	delete(u.usersByNameOrMail, strings.ToLower(user.Name))
	delete(u.usersByNameOrMail, strings.ToLower(user.Mail))
	delete(u.usersByGuid, user.Guid)
	for i, user := range u.config.Users {
		if user.Guid == guid {
			u.config.Users = append(u.config.Users[:i], u.config.Users[i+1:]...)
			break
		}
	}
	return u.Write()
}

func (u *users) GetUser(guid string) (*User, error) {
	user := u.usersByGuid[guid]
	if user == nil {
		return nil, ErrNotFound
	}
	return user, nil
}

func (u *users) GetUserByMail(mail string) (*User, error) {
	user := u.usersByNameOrMail[strings.ToLower(mail)]
	if user == nil {
		return nil, ErrNotFound
	}
	if user.Mail != mail {
		return nil, ErrNotFound
	}
	return user, nil
}

func (u *users) GetUserByResetToken(token string) (*User, error) {
	for _, user := range u.config.Users {
		if user.ResetToken == token && (user.ResetTokenTTL == nil || time.Now().Before(*user.ResetTokenTTL)) {
			return user, nil
		}
	}
	return nil, ErrNotFound
}

func (u *users) StartPasswordReset(mail string) (User, error) {
	user := u.usersByNameOrMail[strings.ToLower(mail)]
	if user == nil {
		return User{}, ErrNotFound
	}

	// Generate 20 random bytes (160 bits) which encodes to 32 base32 characters
	tokenBytes := make([]byte, 20)
	_, err := rand.Read(tokenBytes)
	if err != nil {
		return User{}, err
	}

	token := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(tokenBytes)
	user.ResetToken = token
	ttl := time.Now().Add(24 * time.Hour)
	user.ResetTokenTTL = &ttl

	err = u.Write()
	if err != nil {
		return User{}, err
	}

	return *user, nil
}

func (u *users) PasswordReset(token string, password string) (err error) {
	if token == "" || password == "" {
		return fmt.Errorf("must specify a token and password")
	}

	user, err := u.GetUserByResetToken(token)
	if err != nil {
		return err
	}
	err = user.SetPassword(password)
	if err != nil {
		return err
	}

	user.ResetToken = ""
	user.ResetTokenTTL = nil

	return u.Write()
}

func (u *users) CheckPassword(username, password string) bool {
	user := u.usersByNameOrMail[strings.ToLower(username)]
	if nil == user {
		return false
	}
	err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	return err == nil
}

func (u *users) Groups() []Group {
	groups := make([]Group, 0, len(u.config.Groups))
	for _, group := range u.config.Groups {
		groups = append(groups, *group)
	}
	return groups
}

func (u *users) CheckGroups(names ...string) error {
	for _, name := range names {
		if _, ok := u.groupsByName[name]; !ok {
			return fmt.Errorf("there is no group with name %s", name)
		}
	}
	return nil
}

func (u *users) AddGroup(name string) (Group, error) {
	group := u.groupsByName[name]
	if group != nil {
		return *group, nil
	}

	group = &Group{Name: name}
	group.Guid = uuid.New().String()
	u.groupsByName[name] = group
	u.config.Groups = append(u.config.Groups, group)

	err := u.Write()
	if err != nil {
		return Group{}, err
	}
	return *group, nil
}

func (u *users) DeleteGroup(guid string) error {
	group := u.groupsByGuid[guid]
	if group == nil {
		return ErrNotFound
	}

	for _, user := range u.config.Users {
		for _, groupname := range user.Groups {
			if groupname == group.Name {
				return fmt.Errorf("group %s is still used", group.Name)
			}
		}
	}

	delete(u.groupsByGuid, guid)
	delete(u.groupsByName, group.Name)
	for i, group := range u.config.Groups {
		if group.Guid == guid {
			u.config.Groups = append(u.config.Groups[:i], u.config.Groups[i+1:]...)
			break
		}
	}
	return u.Write()
}
