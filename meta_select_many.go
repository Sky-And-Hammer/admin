package admin

import (
	"errors"
	"html/template"
	"reflect"

	"github.com/Sky-And-Hammer/TM_EC"
	"github.com/Sky-And-Hammer/TM_EC/resource"
	"github.com/Sky-And-Hammer/TM_EC/utils"
)

//	'SelectManyConfig' meta configuration used for select many
type SelectManyConfig struct {
	Collection               interface{}
	SelectionTemplate        string
	SelectMode               string
	Select2ResultTemplate    template.JS
	Select2SelectionTemplate template.JS
	RemoteDataResource       *Resource
	SelectOneConfig
}

//	'GetTemplate' get template for selection template
func (selectManyConfig SelectManyConfig) GetTemplate(context *Context, metaType string) ([]byte, error) {
	if metaType == "form" && selectManyConfig.SelectionTemplate != "" {
		return context.Asset(selectManyConfig.SelectionTemplate)
	}
	return nil, errors.New("not implemented")
}

//	'ConfigureECMeta' configure select many meta
func (selectManyConfig *SelectManyConfig) ConfigureECMeta(metaor resource.Metaor) {
	if meta, ok := metaor.(*Meta); ok {
		selectManyConfig.SelectOneConfig.Collection = selectManyConfig.Collection
		selectManyConfig.SelectOneConfig.SelectMode = selectManyConfig.SelectMode
		selectManyConfig.SelectOneConfig.RemoteDataResource = selectManyConfig.RemoteDataResource

		selectManyConfig.SelectOneConfig.ConfigureECMeta(meta)

		selectManyConfig.RemoteDataResource = selectManyConfig.SelectOneConfig.RemoteDataResource
		meta.Type = "select_many"

		// Set FormattedValuer
		if meta.FormattedValuer == nil {
			meta.SetFormattedValuer(func(record interface{}, context *TM_EC.Context) interface{} {
				reflectValues := reflect.Indirect(reflect.ValueOf(meta.GetValuer()(record, context)))
				var results []string
				for i := 0; i < reflectValues.Len(); i++ {
					results = append(results, utils.Stringify(reflectValues.Index(i).Interface()))
				}
				return results
			})
		}
	}
}
