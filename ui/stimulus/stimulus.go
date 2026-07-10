package stimulus

import "ily.dev/domi"

var (
	Action     = domi.Name("data-action")
	Controller = domi.Name("data-controller")
)

func init() {
	domi.RegisterCombining("data-action", " ")
}

func Target(controller, name string) domi.Attr {
	return domi.Name("data-" + controller + "-target")(name)
}

func Value(controller, name string) func(...string) domi.Attr {
	return domi.Name("data-" + controller + "-" + name + "-value")
}
