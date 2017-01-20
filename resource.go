package admin

import (
	"fmt"
	"net/http"
	"reflect"
	"strings"

	//  The fantastic ORM library for Golang, aims to be developer friendly.
	"github.com/jinzhu/gorm"
	//	a golang library using Common Locale Data Repository to format dates
	"github.com/jinzhu/inflection"

	"github.com/Sky-And-Hammer/TM_EC"
	"github.com/Sky-And-Hammer/TM_EC/resource"
	"github.com/Sky-And-Hammer/TM_EC/utils"
	"github.com/Sky-And-Hammer/roles"
)

//	'Resource' is the most important thing for EC admin, every model is defined as resource, EC admin will genetate management interface based on it's definition
type Resource struct {
	resource.Resource
	Config        *Config
	Metas         []*Meta
	Actions       []*Action
	SearchHandler func(keyword string, context *TM_EC.Context) *gorm.DB

	admin          *Admin
	params         string
	base           *Resource
	scopes         []*Scope
	filters        []*Filter
	searchAttrs    *[]string
	sortableAttrs  *[]string
	indexSections  []*Section
	newSections    []*Section
	editSections   []*Section
	showSections   []*Section
	isSetShowAttrs bool
	cachedMetas    *map[string][]*Meta
}

//	'Meta' register meta for admin resource
func (res *Resource) Meta(meta *Meta) *Meta {
	if res.GetMeta(meta.Name) != nil {
		utils.ExitWithMsg("Duplicated meta %v defined for resource %v", meta.Name, res.Name)
	}
	res.Metas = append(res.Metas, meta)
	meta.baseResource = res
	meta.updateMeta()
	return meta
}

//	'GetAdmin' get admin from resource
func (res Resource) GetAdmin() *Admin {
	return res.admin
}

func (res Resource) GetPrimaryValue(request *http.Request) string {
	if request != nil {
		return request.URL.Query().Get(res.ParamIDName())
	}
	return ""
}

func (res Resource) ParamIDName() string {
	return fmt.Sprintf(":%v_id", inflection.Singular(utils.ToParamString(res.Name)))
}

func (res *Resource) ToParam() string {
	if res.params == "" {
		if value, ok := res.Value.(interface {
			ToParam() string
		}); ok {
			res.params = value.ToParam()
		} else {
			if res.Config.Singleton == true {
				res.params = utils.ToParamString(res.Name)
			}
			res.params = utils.ToParamString(inflection.Plural(res.Name))
		}
	}
	return res.params
}

func (res *Resource) UseTheme(theme interface{}) []ThemeInterface {
	var themeInterface ThemeInterface
	if ti, ok := theme.(ThemeInterface); ok {
		themeInterface = ti
	} else if str, ok := theme.(string); ok {
		themeInterface = Theme{Name: str}
	}

	if themeInterface != nil {
		res.Config.Themes = append(res.Config.Themes, themeInterface)

		// Config Admin Theme
		for _, pth := range themeInterface.GetViewPaths() {
			res.GetAdmin().RegisterViewPath(pth)
		}
		themeInterface.ConfigAdminTheme(res)
	}
	return res.Config.Themes
}

func (res *Resource) GetTheme(name string) ThemeInterface {
	for _, theme := range res.Config.Themes {
		if theme.GetName() == name {
			return theme
		}
	}
	return nil
}

func (res *Resource) Decode(context *TM_EC.Context, value interface{}) error {
	return resource.Decode(context, value, res)
}

func (res *Resource) convertObjectToJSONMap(context *Context, value interface{}, kind string) interface{} {
	reflectValue := reflect.ValueOf(value)
	for reflectValue.Kind() == reflect.Ptr {
		reflectValue = reflectValue.Elem()
	}

	switch reflectValue.Kind() {
	case reflect.Slice:
		values := []interface{}{}
		for i := 0; i < reflectValue.Len(); i++ {
			if reflectValue.Index(i).Kind() == reflect.Ptr {
				values = append(values, res.convertObjectToJSONMap(context, reflectValue.Index(i).Interface(), kind))
			} else {
				values = append(values, res.convertObjectToJSONMap(context, reflectValue.Index(i).Addr().Interface(), kind))
			}
		}
		return values
	case reflect.Struct:
		var metas []*Meta
		if kind == "index" {
			metas = res.ConvertSectionToMetas(res.allowedSections(res.IndexAttrs(), context, roles.Update))
		} else if kind == "edit" {
			metas = res.ConvertSectionToMetas(res.allowedSections(res.EditAttrs(), context, roles.Update))
		} else if kind == "show" {
			metas = res.ConvertSectionToMetas(res.allowedSections(res.ShowAttrs(), context, roles.Read))
		}

		values := map[string]interface{}{}
		for _, meta := range metas {
			if meta.HasPermission(roles.Read, context.Context) {
				if meta.Resource != nil && (meta.FieldStruct != nil && meta.FieldStruct.Relationship != nil && (meta.FieldStruct.Relationship.Kind == "has_one" || meta.FieldStruct.Relationship.Kind == "has_many")) {
					values[meta.GetName()] = meta.Resource.convertObjectToJSONMap(context, context.RawValueOf(value, meta), kind)
				} else {
					values[meta.GetName()] = context.FormattedValueOf(value, meta)
				}
			}
		}
		return values
	default:
		return value
	}
}

func (res *Resource) allAttrs() []string {
	var attrs []string
	scope := &gorm.Scope{Value: res.Value}

Fields:
	for _, field := range scope.GetModelStruct().StructFields {
		for _, meta := range res.Metas {
			if field.Name == meta.FieldName {
				attrs = append(attrs, meta.Name)
				continue Fields
			}
		}

		if field.IsForeignKey {
			continue
		}

		for _, value := range []string{"CreatedAt", "UpdatedAt", "DeletedAt"} {
			if value == field.Name {
				continue Fields
			}
		}

		if (field.IsNormal || field.Relationship != nil) && !field.IsIgnored {
			attrs = append(attrs, field.Name)
			continue
		}

		fieldType := field.Struct.Type
		for fieldType.Kind() == reflect.Ptr || fieldType.Kind() == reflect.Slice {
			fieldType = fieldType.Elem()
		}

		if fieldType.Kind() == reflect.Struct {
			attrs = append(attrs, field.Name)
		}
	}

MetaIncluded:
	for _, meta := range res.Metas {
		for _, attr := range attrs {
			if attr == meta.FieldName || attr == meta.Name {
				continue MetaIncluded
			}
		}
		attrs = append(attrs, meta.Name)
	}

	return attrs
}

func (res *Resource) getAttrs(attrs []string) []string {
	if len(attrs) == 0 {
		return res.allAttrs()
	}

	var onlyExcludeAttrs = true
	for _, attr := range attrs {
		if !strings.HasPrefix(attr, "-") {
			onlyExcludeAttrs = false
			break
		}
	}

	if onlyExcludeAttrs {
		return append(res.allAttrs(), attrs...)
	}
	return attrs
}

func (res *Resource) IndexAttrs(values ...interface{}) []*Section {
	res.setSections(&res.indexSections, values...)
	res.SearchAttrs()
	return res.indexSections
}

func (res *Resource) NewAttrs(values ...interface{}) []*Section {
	res.setSections(&res.newSections, values...)
	return res.newSections
}

func (res *Resource) EditAttrs(values ...interface{}) []*Section {
	res.setSections(&res.editSections, values...)
	return res.editSections
}

func (res *Resource) ShowAttrs(values ...interface{}) []*Section {
	if len(values) > 0 {
		if values[len(values)-1] == false {
			values = values[:len(values)-1]
		} else {
			res.isSetShowAttrs = true
		}
	}
	res.setSections(&res.showSections, values...)
	return res.showSections
}

func (res *Resource) SortableAttrs(columns ...string) []string {
	if len(columns) != 0 || res.sortableAttrs == nil {
		if len(columns) == 0 {
			columns = res.ConvertSectionToStrings(res.indexSections)
		}
		res.sortableAttrs = &[]string{}
		scope := res.GetAdmin().Config.DB.NewScope(res.Value)
		for _, column := range columns {
			if field, ok := scope.FieldByName(column); ok && field.DBName != "" {
				attrs := append(*res.sortableAttrs, column)
				res.sortableAttrs = &attrs
			}
		}
	}
	return *res.sortableAttrs
}

func (res *Resource) SearchAttrs(columns ...string) []string {
	if len(columns) != 0 || res.searchAttrs == nil {
		if len(columns) == 0 {
			columns = res.ConvertSectionToStrings(res.indexSections)
		}

		if len(columns) > 0 {
			res.searchAttrs = &columns
			res.SearchHandler = func(keyword string, context *TM_EC.Context) *gorm.DB {
				var filterFields []filterField
				for _, column := range columns {
					filterFields = append(filterFields, filterField{FieldName: column})
				}
				return filterResourceByFields(res, filterFields, keyword, context.GetDB(), context)
			}
		}
	}

	return columns
}

func (res *Resource) getCachedMetas(cacheKey string, fc func() []resource.Metaor) []*Meta {
	if res.cachedMetas == nil {
		res.cachedMetas = &map[string][]*Meta{}
	}

	if values, ok := (*res.cachedMetas)[cacheKey]; ok {
		return values
	}

	values := fc()
	var metas []*Meta
	for _, value := range values {
		metas = append(metas, value.(*Meta))
	}
	(*res.cachedMetas)[cacheKey] = metas
	return metas
}

func (res *Resource) GetMetas(attrs []string) []resource.Metaor {
	if len(attrs) == 0 {
		attrs = res.allAttrs()
	}
	var showSections, ignoredAttrs []string
	for _, attr := range attrs {
		if strings.HasPrefix(attr, "-") {
			ignoredAttrs = append(ignoredAttrs, strings.TrimLeft(attr, "-"))
		} else {
			showSections = append(showSections, attr)
		}
	}

	primaryKey := res.PrimaryFieldName()

	metas := []resource.Metaor{}

Attrs:
	for _, attr := range showSections {
		for _, a := range ignoredAttrs {
			if attr == a {
				continue Attrs
			}
		}

		var meta *Meta
		for _, m := range res.Metas {
			if m.GetName() == attr {
				meta = m
				break
			}
		}

		if meta == nil {
			meta = &Meta{Name: attr, baseResource: res}
			if attr == primaryKey {
				meta.Type = "hidden"
			}
			meta.updateMeta()
		}

		metas = append(metas, meta)
	}

	return metas
}

//	'GetMeta' get meta with name
func (res *Resource) GetMeta(name string) *Meta {
	for _, meta := range res.Metas {
		if meta.Name == name || meta.GetFieldName() == name {
			return meta
		}
	}
	return nil
}

func (res *Resource) GetMetaOrNew(name string) *Meta {
	if meta := res.GetMeta(name); meta != nil {
		return meta
	}

	if field, ok := res.GetAdmin().Config.DB.NewScope(res.Value).FieldByName(name); ok {
		meta := &Meta{Name: name, baseResource: res}
		if field.IsPrimaryKey {
			meta.Type = "hidden"
		}
		meta.updateMeta()
		res.Metas = append(res.Metas, meta)
		return meta
	}

	return nil
}

func (res *Resource) allowedSections(sections []*Section, context *Context, roles ...roles.PermissionMode) []*Section {
	var newSections []*Section
	for _, section := range sections {
		newSection := Section{Resource: section.Resource, Title: section.Title}
		var editableRows [][]string
		for _, row := range section.Rows {
			var editableColumns []string
			for _, column := range row {
				for _, role := range roles {
					meta := res.GetMetaOrNew(column)
					if meta != nil && meta.HasPermission(role, context.Context) {
						editableColumns = append(editableColumns, column)
						break
					}
				}
			}
			if len(editableColumns) > 0 {
				editableRows = append(editableRows, editableColumns)
			}
		}
		newSection.Rows = editableRows
		newSections = append(newSections, &newSection)
	}
	return newSections
}

func (res *Resource) configure() {
	modelType := utils.ModelType(res.Value)
	for i := 0; i < modelType.NumField(); i++ {
		if fieldStruct := modelType.Field(i); fieldStruct.Anonymous {
			if injector, ok := reflect.New(fieldStruct.Type).Interface().(resource.ConfigureResourceInterface); ok {
				injector.ConfigureECResource(res)
			}
		}
	}

	if injector, ok := res.Value.(resource.ConfigureResourceInterface); ok {
		injector.ConfigureECResource(res)
	}
}
