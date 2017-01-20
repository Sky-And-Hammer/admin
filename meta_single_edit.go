package admin

import (
	"errors"

	"github.com/Sky-And-Hammer/TM_EC/resource"
)

//	'SingleEditConfig' meta configuration used for single edit
type SingleEditConfig struct {
	Template string
	metaConfig
}

//	'GetTemplate' get template for single edit
func (singleEditConfig SingleEditConfig) GetTemplate(context *Context, metaType string) ([]byte, error) {
	if metaType == "form" && singleEditConfig.Template != "" {
		return context.Asset(singleEditConfig.Template)
	}
	return nil, errors.New("not implemented")
}

//	'ConfigureECMeta' configure single edit meta
func (singleEditConfig *SingleEditConfig) ConfigureECMeta(metaor resource.Metaor) {}
