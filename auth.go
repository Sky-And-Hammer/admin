package admin

import (
	"github.com/Sky-And-Hammer/TM_EC"
)

//  'Auth' is an auth interface that used to EC admin
//  if you want to implement an authorization gataway for admin interface, you could implement this interface, and set it to the admin with 'admin.SetAuth(auth)'
type Auth interface {
	GetCurrentUser(*Context) TM_EC.CurrentUser
	LoginURL(*Context) string
	LogoutURL(*Context) string
}
