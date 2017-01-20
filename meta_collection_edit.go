package admin

import (
	"errors"

	"github.com/Sky-And-Hammer/TM_EC/resource"
)

//	'CollectionEditConfig' meta configuration used for collection edit
type CollectionEditConfig struct {
	Template string
	metaConfig
}

//	'GetTemplate' get template for collection edit
func (collecctionEditConfig CollectionEditConfig) GetTemplate(context *Context, metaType string) ([]byte, error) {
	if metaType == "form" && collecctionEditConfig.Template != "" {
		return context.Asset(collecctionEditConfig.Template)
	}
	return nil, errors.New("not implemented")
}

//	'collecctionEditConfig' configure collection edit meta
func (collecctionEditConfig *CollectionEditConfig) ConfigureECMeta(metaor resource.Metaor) {}
