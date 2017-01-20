package admin

import (
	//  The fantastic ORM library for Golang, aims to be developer friendly.
	"github.com/jinzhu/gorm"

	"github.com/Sky-And-Hammer/TM_EC"
)

func (res *Resource) Scope(scope *Scope) {
	if scope.Label == "" {
		scope.Label = scope.Name
	}
	res.scopes = append(res.scopes, scope)
}

//	'Scope' scope definiation
type Scope struct {
	Name    string
	Label   string
	Group   string
	Handle  func(*gorm.DB, *TM_EC.Context) *gorm.DB
	Default bool
}
