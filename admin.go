package admin

import (
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	//  Inflection pluralizes and singularizes English nouns
	"github.com/jinzhu/inflection"
	//	a golang library using Common Locale Data Repository to format dates
	"github.com/theplant/cldr"

	"github.com/Sky-And-Hammer/TM_EC"
	"github.com/Sky-And-Hammer/TM_EC/resource"
	"github.com/Sky-And-Hammer/TM_EC/utils"
)

//	'Admin' is a struct that used to generate admin/api interface
type Admin struct {
	SiteName         string
	Config           *TM_EC.Config
	I18n             I18n
	AssetFS          AssetFSInterface
	menus            []*Menu
	resources        []*Resource
	searchResources  []*Resource
	Auth             Auth
	router           *Router
	funcMaps         template.FuncMap
	metaConfigorMaps map[string]func(*Meta)
}

//	'ResourceNamer' is an interface for models that defined methos 'ResourceName'
type ResourceNamer interface {
	ResourceName() string
}

//	'New' new admin with configuration
func New(config *TM_EC.Config) *Admin {
	admin := Admin{
		Config:           config,
		funcMaps:         make(template.FuncMap),
		router:           newRouter(),
		metaConfigorMaps: metaConfigorMaps,
	}

	admin.SetAssetFS(&AssetFileSystem{})
	admin.registerCompositePrimaryKeyCallback()
	return &admin
}

// SetSiteName set site's name, the name will be used as admin HTML title and admin interface will auto load javascripts, stylesheets files based on its value
// For example, if you named it as `Qor Demo`, admin will look up `qor_demo.js`, `qor_demo.css` in QOR view paths, and load them if found
func (admin *Admin) SetSiteName(siteName string) {
	admin.SiteName = siteName
}

//	'SetAuth' set admin's authorization gateway
func (admin *Admin) SetAuth(auth Auth) {
	admin.Auth = auth
}

//	'SetAssetFS' set AssetFS for admin
func (admin *Admin) SetAssetFS(assetFS AssetFSInterface) {
	admin.AssetFS = assetFS
	globalAssetFSes = append(globalAssetFSes, assetFS)
	admin.AssetFS.RegisterPath(filepath.Join(root, "app/views/ec"))
	for _, viewPath := range globalViewPaths {
		admin.RegisterViewPath(viewPath)
	}
}

//	'RegisterViewPath' register view path for admin
func (admin *Admin) RegisterViewPath(pth string) {
	if admin.AssetFS.RegisterPath(filepath.Join(root, "root", pth)) != nil {
		for _, gopath := range strings.Split(os.Getenv("GOPATH"), ":") {
			if admin.AssetFS.RegisterPath(filepath.Join(gopath, "src", pth)) == nil {
				break
			}
		}
	}
}

//	'RegisterMetaConfigor' register configor for a kind, it will be called when register those kind of metas
func (admin *Admin) RegisterMetaConfigor(king string, fc func(*Meta)) {
	admin.metaConfigorMaps[king] = fc
}

//	'RegisterFuncMap' register view funcs, it could be used in view templates
func (admin *Admin) RegisterFuncMap(name string, fc interface{}) {
	admin.funcMaps[name] = fc
}

//	'GetRouter' get router from admin
func (admin *Admin) GetRouter() *Router {
	return admin.router
}

func (admin *Admin) newResource(value interface{}, config ...*Config) *Resource {
	var configuration *Config
	if len(config) > 0 {
		configuration = config[0]
	}

	if configuration == nil {
		configuration = &Config{}
	}

	res := &Resource{
		Resource:    *resource.New(value),
		Config:      configuration,
		cachedMetas: &map[string][]*Meta{},
		admin:       admin,
	}

	res.Permission = configuration.Permission
	if configuration.Name != "" {
		res.Name = configuration.Name
	} else if namer, ok := value.(ResourceNamer); ok {
		res.Name = namer.ResourceName()
	}

	modelType := utils.ModelType(res.Value)
	for i := 0; i < modelType.NumField(); i++ {
		if filedStruct := modelType.Field(i); filedStruct.Anonymous {
			if injector, ok := reflect.New(filedStruct.Type).Interface().(resource.ConfigureResourceBeforeInitializeInterface); ok {
				injector.ConfigureECResourceBeforeInitialize(res)

			}
		}
	}

	if injector, ok := res.Value.(resource.ConfigureResourceBeforeInitializeInterface); ok {
		injector.ConfigureECResourceBeforeInitialize(res)
	}

	findOneHandler := res.FindOneHandler
	res.FindOneHandler = func(result interface{}, metaValues *resource.MetaValues, context *TM_EC.Context) error {
		if context.ResourceID == "" {
			context.ResourceID = res.GetPrimaryValue(context.Request)
		}
		return findOneHandler(result, metaValues, context)
	}

	res.UseTheme("slideout")
	return res
}

//	'NewResource' initialize a new ec resource, won't add it to admin, just initialize it
func (admin *Admin) NewResource(value interface{}, config ...*Config) *Resource {
	res := admin.newResource(value, config...)
	res.configure()
	return res
}

//	'AddResource' make a model manageable from admin interface
func (admin *Admin) AddResource(value interface{}, config ...*Config) *Resource {
	res := admin.newResource(value, config...)
	if !res.Config.Invisible {
		var menuName string
		if res.Config.Singleton {
			menuName = res.Name
		} else {
			menuName = inflection.Plural(res.Name)
		}

		menu := &Menu{rawPath: res.ToParam(), Name: menuName, Permission: res.Config.Permission, Priority: res.Config.Priority}
		admin.menus = appendMenu(admin.menus, res.Config.Menu, menu)
		res.Action(&Action{
			Name:   "Delete",
			Method: "DELETE",
			URL: func(record interface{}, context *Context) string {
				return context.URLFor(record, res)
			},
			Permission: res.Config.Permission,
			Modes:      []string{"menu_item"},
		})
	}

	admin.resources = append(admin.resources, res)
	if admin.router.Mounted() {
		admin.generateMenuLinks()
		res.configure()
		if !res.Config.Invisible {
			admin.RegisterResourceRouters(res, "create", "updata", "read", "delete")
		}
	}

	return res
}

//	'GetResources' get defined resource from admin
func (admin *Admin) GetResources() []*Resource {
	return admin.resources
}

//	'GetResource' get resource with name
func (admin *Admin) GetResource(name string) (resource *Resource) {
	for _, res := range admin.resources {
		modelType := utils.ModelType(res.Value)
		if res.ToParam() == name || res.Name == name || modelType.String() == name {
			return res
		}

		if modelType.Name() == name {
			resource = res
		}
	}

	return
}

func (admin *Admin) AddSearchResrouce(resources ...*Resource) {
	admin.searchResources = append(admin.searchResources, resources...)
}

func (admin *Admin) GetSearchResrouces() []*Resource {
	return admin.searchResources
}

type I18n interface {
	Scope(scope string) I18n
	Default(value string) I18n
	T(locale string, key string, args ...interface{}) template.HTML
}

func (admin *Admin) T(context *TM_EC.Context, key string, value string, values ...interface{}) template.HTML {
	locale := utils.GetLocale(context)
	if admin.I18n == nil {
		if result, err := cldr.Parse(locale, value, values...); err == nil {
			return template.HTML(result)
		}
		return template.HTML(key)
	}

	return admin.I18n.Default(value).T(locale, key, values...)
}
