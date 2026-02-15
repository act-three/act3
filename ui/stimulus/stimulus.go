package stimulus

import (
	"ily.dev/act3/html/attr"
)

var (
	Action     = attr.Attr("data-action")
	Controller = attr.Attr("data-controller")
)

func init() {
	attr.RegisterCombining("data-action")
}

func Target(controller, name string) attr.Node {
	return attr.Attr("data-" + controller + "-target")(name)
}

func Value(controller, name string) attr.AttrName {
	return attr.Attr("data-" + controller + "-" + name + "-value")
}
