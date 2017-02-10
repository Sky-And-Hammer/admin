package admin_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	//  The fantastic ORM library for Golang, aims to be developer friendly.
	"github.com/jinzhu/gorm"

	// "github.com/mattn/go-sqlite3"

	"github.com/Sky-And-Hammer/TM_EC"
	"github.com/Sky-And-Hammer/TM_EC/test/utils"
	"github.com/Sky-And-Hammer/admin"
	"github.com/Sky-And-Hammer/media_library"
	// "github.com/qor/media_library"
)

type CreditCard struct {
	gorm.Model
	UserID uint
	Number string
	Issure string
}

type Address struct {
	gorm.Model
	UserID   uint
	Address1 string
	Address2 string
}

type Language struct {
	gorm.Model
	Name string
}

type User struct {
	gorm.Model
	Name         string
	Age          uint
	Role         string
	Active       bool
	RegisteredAt *time.Time
	Avatar       media_library.FileSystem
	CreditCard   CreditCard
	Addresses    []Address
	Language     []Language `gorm:"many2many:user_languages;"`

	Profile Profile
}

type Profile struct {
	gorm.Model
	UserID uint
	Name   string
	Sex    string

	Phone Phone
}

type Phone struct {
	gorm.Model

	ProfileID uint64
	Num       string
}

var (
	server *httptest.Server
	db     *gorm.DB
	Admin  *admin.Admin
)

func init() {
	mux := http.NewServeMux()
	db = utils.TestDB()
	models := []interface{}{&User{}, &CreditCard{}, &Address{}, &Language{}, &Profile{}, &Phone{}}
	for _, value := range models {
		db.DropTableIfExists(value)
		db.AutoMigrate(value)
	}

	Admin = admin.New(&TM_EC.Config{DB: db})
	user := Admin.AddResource(&User{})
	user.Meta(&admin.Meta{Name: "Language", Type: "select_many",
		Collection: func(resource interface{}, context *TM_EC.Context) (results [][]string) {
			if languages := []Language{}; !context.GetDB().Find(&languages).RecordNotFound() {
				for _, language := range languages {
					results = append(results, []string{fmt.Sprint(language.ID), language.Name})
				}
			}
			return
		}})
	Admin.MountTo("/admin", mux)
	server = httptest.NewServer(mux)
}
